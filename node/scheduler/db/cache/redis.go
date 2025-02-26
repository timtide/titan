package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/fatih/structs"
	"github.com/linguohua/titan/api"

	"github.com/go-redis/redis/v8"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	// redisKeyWaitingDataTaskList  server name
	redisKeyWaitingDataTaskList = "Titan:WaitingDataTaskList:%s"
	// redisKeyRunningDataTaskList  server name
	redisKeyRunningDataTaskList = "Titan:RunningDataTaskList:%s"
	// redisKeyRunningDataTask  server name:cid
	redisKeyRunningDataTask = "Titan:RunningDataTask:%s:%s"
	// redisKeyCacheResult  server name
	redisKeyCacheResult = "Titan:CacheResult:%s"
	// redisKeyNodeBlockFid  deviceID
	redisKeyNodeBlockFid = "Titan:NodeBlockFid:%s"
	// redisKeyValidatorList server name
	redisKeyValidatorList = "Titan:ValidatorList:%s"
	// redisKeyValidateRoundID server name
	redisKeyValidateRoundID = "Titan:ValidateRoundID:%s"
	// redisKeyVerifyingList server name
	redisKeyVerifyingList = "Titan:VerifyingList:%s"
	// redisKeyNodeInfo  deviceID
	redisKeyNodeInfo = "Titan:NodeInfo:%s"
	// redisKeyBlockDownloadRecord serial number
	redisKeyBlockDownloadRecord = "Titan:BlockDownloadRecord:%d"
	// redisKeyBlockDownloadSN
	redisKeyBlockDownloadSN       = "Titan:BlockDownloadRecordSN"
	redisKeyCarfileLatestDownload = "Titan:LatestDownload:%s"

	// redisKeyCacheErrorList server name cacheID
	redisKeyCacheErrorList = "Titan:CacheErrorList:%s:%s"

	// redisKeyBaseInfo  server name
	redisKeyBaseInfo = "Titan:BaseInfo:%s"
)

const cacheErrorExpiration = 72 //hour

// TypeRedis redis
func TypeRedis() string {
	return "Redis"
}

type redisDB struct {
	cli *redis.Client
}

// InitRedis init redis pool
func InitRedis(url string) (DB, error) {
	// fmt.Printf("redis init url:%v", url)

	redisDB := &redisDB{redis.NewClient(&redis.Options{
		Addr:      url,
		Dialer:    nil,
		OnConnect: nil,
	})}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := redisDB.cli.Ping(ctx).Result()

	return redisDB, err
}

// cache result info
func (rd redisDB) SetCacheResultInfo(info *api.CacheResultInfo) (int64, error) {
	key := fmt.Sprintf(redisKeyCacheResult, serverName)

	bytes, err := json.Marshal(info)
	if err != nil {
		return 0, err
	}

	return rd.cli.RPush(context.Background(), key, bytes).Result()
}

func (rd redisDB) GetCacheResultInfo() (*api.CacheResultInfo, error) {
	key := fmt.Sprintf(redisKeyCacheResult, serverName)

	// value, err := rd.cli.LIndex(context.Background(), key, 0).Result()
	value, err := rd.cli.LPop(context.Background(), key).Result()

	if value == "" {
		return nil, nil
	}

	var info api.CacheResultInfo
	bytes, err := redigo.Bytes(value, nil)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bytes, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

func (rd redisDB) GetCacheResultNum() int64 {
	key := fmt.Sprintf(redisKeyCacheResult, serverName)

	l, _ := rd.cli.LLen(context.Background(), key).Result()
	return l
}

// running data list
func (rd redisDB) SetDataTaskToRunningList(hash, cacheID string) error {
	key := fmt.Sprintf(redisKeyRunningDataTaskList, serverName)

	info := DataTask{CarfileHash: hash, CacheID: cacheID}
	bytes, err := json.Marshal(info)
	if err != nil {
		return err
	}
	key2 := fmt.Sprintf(redisKeyRunningDataTask, serverName, hash)

	ctx := context.Background()
	_, err = rd.cli.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		pipeliner.SAdd(context.Background(), key, bytes)
		// Expire
		pipeliner.Set(context.Background(), key2, cacheID, time.Second*time.Duration(15))

		return nil
	})
	return err
}

func (rd redisDB) GetDataTasksWithRunningList() ([]*DataTask, error) {
	key := fmt.Sprintf(redisKeyRunningDataTaskList, serverName)

	values, err := rd.cli.SMembers(context.Background(), key).Result()
	if err != nil || values == nil {
		return nil, err
	}

	list := make([]*DataTask, 0)
	for _, value := range values {
		var info DataTask
		bytes, err := redigo.Bytes(value, nil)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(bytes, &info); err != nil {
			continue
		}

		list = append(list, &info)
	}

	return list, nil
}

// running data details
func (rd redisDB) SetRunningDataTask(hash, cacheID string, timeout int64) error {
	key := fmt.Sprintf(redisKeyRunningDataTask, serverName, hash)
	// Expire
	_, err := rd.cli.Set(context.Background(), key, cacheID, time.Second*time.Duration(timeout)).Result()
	return err
}

func (rd redisDB) GetRunningDataTaskExpiredTime(hash string) (time.Duration, error) {
	key := fmt.Sprintf(redisKeyRunningDataTask, serverName, hash)
	// Expire
	return rd.cli.TTL(context.Background(), key).Result()
}

func (rd redisDB) GetRunningDataTask(hash string) (string, error) {
	key := fmt.Sprintf(redisKeyRunningDataTask, serverName, hash)

	return rd.cli.Get(context.Background(), key).Result()
}

func (rd redisDB) RemoveRunningDataTask(hash, cacheID string) error {
	key := fmt.Sprintf(redisKeyRunningDataTask, serverName, hash)
	key2 := fmt.Sprintf(redisKeyRunningDataTaskList, serverName)

	ctx := context.Background()
	_, err := rd.cli.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		pipeliner.Del(context.Background(), key)

		info := DataTask{CarfileHash: hash, CacheID: cacheID}
		bytes, err := json.Marshal(info)
		if err != nil {
			return err
		}

		pipeliner.SRem(context.Background(), key2, bytes)

		return nil
	})

	return err
}

// waiting data list
func (rd redisDB) SetWaitingDataTask(info *api.DataInfo) error {
	key := fmt.Sprintf(redisKeyWaitingDataTaskList, serverName)

	bytes, err := json.Marshal(info)
	if err != nil {
		return err
	}

	_, err = rd.cli.RPush(context.Background(), key, bytes).Result()
	return err
}

func (rd redisDB) GetWaitingDataTask(index int64) (*api.DataInfo, error) {
	key := fmt.Sprintf(redisKeyWaitingDataTaskList, serverName)

	value, err := rd.cli.LIndex(context.Background(), key, index).Result()

	if value == "" {
		return nil, redis.Nil
	}

	var info api.DataInfo
	bytes, err := redigo.Bytes(value, nil)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bytes, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

func (rd redisDB) RemoveWaitingDataTasks(infos []*api.DataInfo) error {
	key := fmt.Sprintf(redisKeyWaitingDataTaskList, serverName)

	ctx := context.Background()
	_, err := rd.cli.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		for _, info := range infos {
			bytes, err := json.Marshal(info)
			if err != nil {
				continue
			}

			pipeliner.LRem(context.Background(), key, 1, bytes)
		}

		return nil
	})

	return err
}

// validate round id ++1
func (rd redisDB) IncrValidateRoundID() (int64, error) {
	key := fmt.Sprintf(redisKeyValidateRoundID, serverName)

	return rd.cli.IncrBy(context.Background(), key, 1).Result()
}

// verifying node list
func (rd redisDB) SetNodeToVerifyingList(deviceID string) error {
	key := fmt.Sprintf(redisKeyVerifyingList, serverName)

	_, err := rd.cli.SAdd(context.Background(), key, deviceID).Result()
	return err
}

func (rd redisDB) GetNodesWithVerifyingList() ([]string, error) {
	key := fmt.Sprintf(redisKeyVerifyingList, serverName)

	return rd.cli.SMembers(context.Background(), key).Result()
}

func (rd redisDB) CountVerifyingNode(ctx context.Context) (int64, error) {
	key := fmt.Sprintf(redisKeyVerifyingList, serverName)
	return rd.cli.SCard(ctx, key).Result()
}

func (rd redisDB) RemoveNodeWithVerifyingList(deviceID string) error {
	key := fmt.Sprintf(redisKeyVerifyingList, serverName)

	_, err := rd.cli.SRem(context.Background(), key, deviceID).Result()
	return err
}

func (rd redisDB) RemoveVerifyingList() error {
	key := fmt.Sprintf(redisKeyVerifyingList, serverName)

	_, err := rd.cli.Del(context.Background(), key).Result()
	return err
}

// validator list
func (rd redisDB) SetValidatorsToList(deviceIDs []string, expiration time.Duration) error {
	key := fmt.Sprintf(redisKeyValidatorList, serverName)

	// _, err := rd.cli.SAdd(context.Background(), key, deviceIDs).Result()

	ctx := context.Background()
	_, err := rd.cli.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		pipeliner.Del(context.Background(), key)

		if len(deviceIDs) > 0 {
			pipeliner.SAdd(context.Background(), key, deviceIDs)
			// Expire
			pipeliner.PExpire(context.Background(), key, expiration)
		}

		return nil
	})

	return err
}

func (rd redisDB) GetValidatorsWithList() ([]string, error) {
	key := fmt.Sprintf(redisKeyValidatorList, serverName)

	return rd.cli.SMembers(context.Background(), key).Result()
}

func (rd redisDB) GetValidatorsAndExpirationTime() ([]string, time.Duration, error) {
	key := fmt.Sprintf(redisKeyValidatorList, serverName)

	expiration, err := rd.cli.TTL(context.Background(), key).Result()
	if err != nil {
		return nil, expiration, err
	}

	list, err := rd.cli.SMembers(context.Background(), key).Result()
	if err != nil {
		return nil, expiration, err
	}

	return list, expiration, nil
}

// device info
func (rd redisDB) IncrNodeOnlineTime(deviceID string, onlineTime int64) (float64, error) {
	key := fmt.Sprintf(redisKeyNodeInfo, deviceID)

	mTime := float64(onlineTime) / 60 // second to minute

	return rd.cli.HIncrByFloat(context.Background(), key, "OnlineTime", mTime).Result()
}

func (rd redisDB) SetDeviceInfo(info *api.DevicesInfo) error {
	key := fmt.Sprintf(redisKeyNodeInfo, info.DeviceId)

	ctx := context.Background()
	exist, err := rd.cli.Exists(ctx, key).Result()
	if err != nil {
		return err
	}

	if exist == 1 {
		// update some value
		return rd.updateDeviceInfos(info)
	}

	// _, err = rd.cli.HMSet(ctx, key, structs.Map(info)).Result()
	// if err != nil {
	// 	return err
	// }

	_, err = rd.cli.Pipelined(ctx, func(pipeline redis.Pipeliner) error {
		for field, value := range toMap(info) {
			pipeline.HSet(ctx, key, field, value)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (rd redisDB) updateDeviceInfos(info *api.DevicesInfo) error {
	key := fmt.Sprintf(redisKeyNodeInfo, info.DeviceId)

	m := make(map[string]interface{})
	m["DiskSpace"] = info.DiskSpace

	//TODO
	m["Longitude"] = info.Longitude
	m["Latitude"] = info.Latitude

	_, err := rd.cli.HMSet(context.Background(), key, m).Result()
	return err
}

func (rd redisDB) GetDeviceInfo(deviceID string) (*api.DevicesInfo, error) {
	key := fmt.Sprintf(redisKeyNodeInfo, deviceID)

	var info api.DevicesInfo
	err := rd.cli.HGetAll(context.Background(), key).Scan(&info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

func (rd redisDB) UpdateDeviceInfo(deviceID, field string, value interface{}) error {
	key := fmt.Sprintf(redisKeyNodeInfo, deviceID)

	_, err := rd.cli.HSet(context.Background(), key, field, value).Result()
	if err != nil {
		return err
	}

	return err
}

func (rd redisDB) UpdateDevicesInfo(field string, values map[string]interface{}) error {
	ctx := context.Background()
	_, err := rd.cli.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		for deviceID, value := range values {
			key := fmt.Sprintf(redisKeyNodeInfo, deviceID)
			pipeliner.HSet(context.Background(), key, field, value)
		}

		return nil
	})

	return err
}

func (rd redisDB) IncrByDeviceInfo(deviceID, field string, value int64) error {
	key := fmt.Sprintf(redisKeyNodeInfo, deviceID)

	_, err := rd.cli.HIncrBy(context.Background(), key, field, value).Result()
	if err != nil {
		return err
	}

	return err
}

func (rd redisDB) IncrByDevicesInfo(field string, values map[string]int64) error {
	ctx := context.Background()
	_, err := rd.cli.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		for deviceID, value := range values {
			key := fmt.Sprintf(redisKeyNodeInfo, deviceID)
			pipeliner.HIncrBy(context.Background(), key, field, value)
		}

		return nil
	})

	return err
}

func (rd redisDB) UpdateNodeCacheBlockInfo(toDeviceID, fromDeviceID string, blockSize int) error {
	//UpdateNodeCacheBlockInfo update node cache block info
	size := int64(blockSize)

	toKey := fmt.Sprintf(redisKeyNodeInfo, toDeviceID)

	ctx := context.Background()
	_, err := rd.cli.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {

		pipeliner.HIncrBy(context.Background(), toKey, BlockCountField, 1)
		pipeliner.HIncrBy(context.Background(), toKey, TotalDownloadField, size)

		if fromDeviceID != "" {
			fromKey := fmt.Sprintf(redisKeyNodeInfo, fromDeviceID)
			pipeliner.HIncrBy(context.Background(), fromKey, DownloadCountField, 1)
			pipeliner.HIncrBy(context.Background(), fromKey, TotalUploadField, size)
		}

		return nil
	})

	return err
}

// download info
func (rd redisDB) SetDownloadBlockRecord(record *DownloadBlockRecord) error {
	ctx := context.Background()
	key := fmt.Sprintf(redisKeyBlockDownloadRecord, record.SN)
	_, err := rd.cli.HMSet(ctx, key, structs.Map(record)).Result()
	if err != nil {
		return err
	}
	_, err = rd.cli.Expire(ctx, key, time.Duration(record.Timeout)*time.Second).Result()
	if err != nil {
		return err
	}

	return nil
}

func (rd redisDB) GetDownloadBlockRecord(sn int64) (*DownloadBlockRecord, error) {
	key := fmt.Sprintf(redisKeyBlockDownloadRecord, sn)

	var record DownloadBlockRecord

	err := rd.cli.HGetAll(context.Background(), key).Scan(&record)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (rd redisDB) RemoveDownloadBlockRecord(sn int64) error {
	key := fmt.Sprintf(redisKeyBlockDownloadRecord, sn)
	_, err := rd.cli.Del(context.Background(), key).Result()
	return err
}

func (rd redisDB) IncrBlockDownloadSN() (int64, error) {
	// node cache tag ++1
	n, err := rd.cli.IncrBy(context.Background(), redisKeyBlockDownloadSN, 1).Result()
	if n >= math.MaxInt64 {
		rd.cli.Set(context.Background(), redisKeyBlockDownloadSN, 0, 0).Result()
	}

	return n, err
}

// latest data of download
func (rd redisDB) AddLatestDownloadCarfile(carfileCID string, userIP string) error {
	maxCount := 5
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key := fmt.Sprintf(redisKeyCarfileLatestDownload, userIP)

	err := rd.cli.ZAdd(ctx, key, &redis.Z{Score: float64(time.Now().Unix()), Member: carfileCID}).Err()
	if err != nil {
		return err
	}

	count, err := rd.cli.ZCard(ctx, key).Result()
	if err != nil {
		return err
	}

	if count > int64(maxCount) {
		err = rd.cli.ZRemRangeByRank(ctx, key, 0, count-int64(maxCount)-1).Err()
		if err != nil {
			return err
		}
	}

	return rd.cli.Expire(ctx, key, 24*time.Hour).Err()
}

func (rd redisDB) GetLatestDownloadCarfiles(userIP string) ([]string, error) {
	key := fmt.Sprintf(redisKeyCarfileLatestDownload, userIP)

	members, err := rd.cli.ZRevRange(context.Background(), key, 0, -1).Result()
	if err != nil {
		return []string{}, err
	}
	return members, nil
}

// cache error details
func (rd redisDB) SaveCacheErrors(cacheID string, infos []*api.CacheError, isClean bool) error {
	key := fmt.Sprintf(redisKeyCacheErrorList, serverName, cacheID)

	ctx := context.Background()
	_, err := rd.cli.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		if isClean {
			pipeliner.Del(context.Background(), key)
		}

		for _, info := range infos {
			bytes, err := json.Marshal(info)
			if err != nil {
				continue
			}
			pipeliner.SAdd(context.Background(), key, bytes)
		}
		// Expire
		pipeliner.PExpire(context.Background(), key, time.Hour*time.Duration(cacheErrorExpiration))

		return nil
	})

	return err
}

func (rd redisDB) GetCacheErrors(cacheID string) ([]*api.CacheError, error) {
	key := fmt.Sprintf(redisKeyCacheErrorList, serverName, cacheID)

	values, err := rd.cli.SMembers(context.Background(), key).Result()
	if err != nil || values == nil {
		return nil, err
	}

	list := make([]*api.CacheError, 0)
	for _, value := range values {
		var info api.CacheError
		bytes, err := redigo.Bytes(value, nil)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(bytes, &info); err != nil {
			continue
		}

		list = append(list, &info)
	}

	return list, nil
}

func (rd redisDB) NodeDownloadCount(deviceID string, blockDownnloadInfo *api.BlockDownloadInfo) error {
	nodeKey := fmt.Sprintf(redisKeyNodeInfo, deviceID)

	ctx := context.Background()
	_, err := rd.cli.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {

		pipeliner.HIncrBy(context.Background(), nodeKey, DownloadCountField, 1)
		pipeliner.HIncrBy(context.Background(), nodeKey, TotalUploadField, int64(blockDownnloadInfo.BlockSize))

		// count carfile download
		if blockDownnloadInfo.BlockCID == blockDownnloadInfo.CarfileCID {
			key := fmt.Sprintf(redisKeyBaseInfo, serverName)
			pipeliner.HIncrBy(context.Background(), key, DownloadCountField, 1)
		}

		return nil
	})

	return err
}

// IsNilErr Is NilErr
func (rd redisDB) IsNilErr(err error) bool {
	return errors.Is(err, redis.Nil)
}

func toMap(info *api.DevicesInfo) map[string]interface{} {
	out := make(map[string]interface{})
	t := reflect.TypeOf(*info)
	v := reflect.ValueOf(*info)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		redisTag := field.Tag.Get("redis")
		if redisTag == "" {
			continue
		}
		out[redisTag] = v.Field(i).Interface()
	}
	return out
}

// node fid of cache ++1
func (rd redisDB) IncrNodeCacheFid(deviceID string, num int) (int, error) {
	key := fmt.Sprintf(redisKeyNodeBlockFid, deviceID)
	n, err := rd.cli.IncrBy(context.Background(), key, int64(num)).Result()
	return int(n), err
}

func (rd redisDB) GetNodeCacheFid(deviceID string) (int64, error) {
	key := fmt.Sprintf(redisKeyNodeBlockFid, deviceID)
	val, err := rd.cli.Get(context.Background(), key).Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// system base info
func (rd redisDB) UpdateBaseInfo(field string, value interface{}) error {
	key := fmt.Sprintf(redisKeyBaseInfo, serverName)

	_, err := rd.cli.HSet(context.Background(), key, field, value).Result()
	if err != nil {
		return err
	}

	return err
}

func (rd redisDB) IncrByBaseInfo(field string, value int64) error {
	key := fmt.Sprintf(redisKeyBaseInfo, serverName)

	_, err := rd.cli.HIncrBy(context.Background(), key, field, value).Result()
	if err != nil {
		return err
	}

	return err
}

func (rd redisDB) GetBaseInfo() (*api.BaseInfo, error) {
	key := fmt.Sprintf(redisKeyBaseInfo, serverName)

	var info api.BaseInfo
	err := rd.cli.HGetAll(context.Background(), key).Scan(&info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}
