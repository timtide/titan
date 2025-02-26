package data

import (
	"sync"
	"time"

	"github.com/linguohua/titan/api"
	"github.com/linguohua/titan/node/scheduler/db/cache"
	"github.com/linguohua/titan/node/scheduler/db/persistent"
	"github.com/linguohua/titan/node/scheduler/node"
	"golang.org/x/xerrors"
)

// Data Data
type Data struct {
	nodeManager *node.Manager
	dataManager *Manager

	carfileCid      string
	carfileHash     string
	reliability     int
	needReliability int
	cacheCount      int
	totalSize       int
	totalBlocks     int
	nodes           int
	expiredTime     time.Time

	CacheMap sync.Map
}

func newData(nodeManager *node.Manager, dataManager *Manager, cid, hash string, reliability int) *Data {
	return &Data{
		nodeManager:     nodeManager,
		dataManager:     dataManager,
		carfileCid:      cid,
		reliability:     0,
		needReliability: reliability,
		cacheCount:      0,
		totalBlocks:     1,
		carfileHash:     hash,
		// CacheMap:        new(sync.Map),
	}
}

func loadData(hash string, dataManager *Manager) *Data {
	dInfo, err := persistent.GetDB().GetDataInfo(hash)
	if err != nil && !persistent.GetDB().IsNilErr(err) {
		log.Errorf("loadData %s err :%s", hash, err.Error())
		return nil
	}
	if dInfo != nil {
		data := &Data{}
		data.carfileCid = dInfo.CarfileCid
		data.nodeManager = dataManager.nodeManager
		data.dataManager = dataManager
		data.totalSize = dInfo.TotalSize
		data.needReliability = dInfo.NeedReliability
		data.reliability = dInfo.Reliability
		data.cacheCount = dInfo.CacheCount
		data.totalBlocks = dInfo.TotalBlocks
		data.nodes = dInfo.Nodes
		data.expiredTime = dInfo.ExpiredTime
		data.carfileHash = dInfo.CarfileHash
		// data.CacheMap = new(sync.Map)

		caches, err := persistent.GetDB().GetCachesWithData(hash)
		if err != nil {
			log.Errorf("loadData hash:%s, GetCachesWithData err:%s", hash, err.Error())
			return data
		}

		for _, cacheID := range caches {
			if cacheID == "" {
				continue
			}
			c := loadCache(cacheID, data)
			if c == nil {
				continue
			}

			data.CacheMap.Store(cacheID, c)
		}

		return data
	}

	return nil
}

func (d *Data) existRootCache() bool {
	exist := false

	d.CacheMap.Range(func(key, value interface{}) bool {
		if exist {
			return true
		}

		if value != nil {
			c := value.(*Cache)
			if c != nil {
				exist = c.isRootCache && c.status == api.CacheStatusSuccess
			}
		}

		return true
	})

	return exist
}

func (d *Data) updateAndSaveCacheingInfo(blockInfo *api.BlockInfo, cache *Cache, createBlocks []*api.BlockInfo) error {
	if !d.existRootCache() {
		d.totalSize = cache.totalSize
		d.totalBlocks = cache.totalBlocks
	}

	dInfo := &api.DataInfo{
		CarfileHash: d.carfileHash,
		TotalSize:   d.totalSize,
		TotalBlocks: d.totalBlocks,
		Reliability: d.reliability,
		CacheCount:  d.cacheCount,
	}

	cInfo := &api.CacheInfo{
		// ID:          cache.dbID,
		CarfileHash: cache.carfileHash,
		CacheID:     cache.cacheID,
		DoneSize:    cache.doneSize,
		Status:      cache.status,
		DoneBlocks:  cache.doneBlocks,
		Reliability: cache.reliability,
		TotalSize:   cache.totalSize,
		TotalBlocks: cache.totalBlocks,
	}

	return persistent.GetDB().SaveCacheingResults(dInfo, cInfo, blockInfo, createBlocks)
}

func (d *Data) updateNodeDiskUsage(nodes []string) {
	values := make(map[string]interface{})

	for _, deviceID := range nodes {
		e := d.nodeManager.GetEdgeNode(deviceID)
		if e != nil {
			values[e.DeviceId] = e.DiskUsage
			continue
		}

		c := d.nodeManager.GetCandidateNode(deviceID)
		if c != nil {
			values[c.DeviceId] = c.DiskUsage
			continue
		}
	}

	err := cache.GetDB().UpdateDevicesInfo(cache.DiskUsageField, values)
	if err != nil {
		log.Errorf("updateNodeDiskUsage err:%s", err.Error())
	}
}

func (d *Data) updateAndSaveCacheEndInfo(doneCache *Cache) error {
	if doneCache.status == api.CacheStatusSuccess {
		d.reliability += doneCache.reliability

		err := cache.GetDB().IncrByBaseInfo(cache.CarFileCountField, 1)
		if err != nil {
			log.Errorf("updateAndSaveCacheEndInfo IncrByBaseInfo err: %s", err.Error())
		}
	}

	dNodes, cNodes := persistent.GetDB().GetNodesFromDataCache(d.carfileHash, doneCache.cacheID)
	if dNodes != nil && len(dNodes) > 0 {
		d.nodes = len(dNodes)
		d.updateNodeDiskUsage(dNodes)
	}
	if cNodes != nil && len(cNodes) > 0 {
		doneCache.nodes = len(cNodes)
	}

	dInfo := &api.DataInfo{
		CarfileHash: d.carfileHash,
		TotalSize:   d.totalSize,
		TotalBlocks: d.totalBlocks,
		Reliability: d.reliability,
		CacheCount:  d.cacheCount,
		Nodes:       d.nodes,
	}

	cInfo := &api.CacheInfo{
		CarfileHash: doneCache.carfileHash,
		CacheID:     doneCache.cacheID,
		Status:      doneCache.status,
		Reliability: doneCache.reliability,
		TotalSize:   doneCache.totalSize,
		TotalBlocks: doneCache.totalBlocks,
		Nodes:       doneCache.nodes,
	}

	return persistent.GetDB().SaveCacheEndResults(dInfo, cInfo)
}

func (d *Data) dispatchCache(cache *Cache) error {
	var err error
	var list map[string]string

	if cache != nil {
		cache.updateCacheInfo()

		list, err = persistent.GetDB().GetUndoneBlocks(cache.cacheID)
		if err != nil {
			return err
		}

	} else {
		var blockID string
		cache, blockID, err = newCache(d, !d.existRootCache())
		if err != nil {
			return err
		}

		d.CacheMap.Store(cache.cacheID, cache)

		list = map[string]string{d.carfileCid: blockID}
	}

	d.cacheCount++

	err = cache.startCache(list)
	if err != nil {
		return err
	}

	return nil
}

func (d *Data) cacheEnd(doneCache *Cache, isContinue bool) {
	var err error

	defer func() {
		if err != nil {
			d.dataManager.recordTaskEnd(d.carfileCid, d.carfileHash, err.Error())
		}
	}()

	err = d.updateAndSaveCacheEndInfo(doneCache)
	if err != nil {
		err = xerrors.Errorf("updateAndSaveCacheEndInfo err:%s", err.Error())
		return
	}

	if !isContinue {
		err = xerrors.Errorf("do not continue")
		return
	}

	if d.cacheCount > d.needReliability {
		err = xerrors.Errorf("cacheCount:%d reach needReliability:%d", d.cacheCount, d.needReliability)
		return
	}

	if d.needReliability <= d.reliability {
		err = xerrors.Errorf("reliability is enough:%d/%d", d.reliability, d.needReliability)
		return
	}

	err = d.dispatchCache(d.getUndoneCache())
}

func (d *Data) getUndoneCache() *Cache {
	// old cache
	var oldCache *Cache
	var oldRootCache *Cache

	d.CacheMap.Range(func(key, value interface{}) bool {
		c := value.(*Cache)

		if c.status != api.CacheStatusSuccess {
			oldCache = c

			if c.isRootCache {
				oldRootCache = c
			}
		}

		return true
	})

	if oldRootCache != nil {
		return oldRootCache
	}

	return oldCache
}

// GetCarfileCid get carfile cid
func (d *Data) GetCarfileCid() string {
	return d.carfileCid
}

// GetCarfileHash get carfile hash
func (d *Data) GetCarfileHash() string {
	return d.carfileHash
}

// GetTotalSize get total size
func (d *Data) GetTotalSize() int {
	return d.totalSize
}

// GetNeedReliability get need reliability
func (d *Data) GetNeedReliability() int {
	return d.needReliability
}

// GetReliability get reliability
func (d *Data) GetReliability() int {
	return d.reliability
}

// GetTotalBlocks get total blocks
func (d *Data) GetTotalBlocks() int {
	return d.totalBlocks
}

// GetTotalNodes get total nodes
func (d *Data) GetTotalNodes() int {
	return d.nodes
}
