// //go:build !windows && !darwin
// // +build !windows,!darwin

package blockstore

// import (
// 	"bytes"
// 	"errors"

// 	"github.com/ipfs/go-datastore"
// 	"github.com/linguohua/titan/node/fsutil"
// 	"github.com/tecbot/gorocksdb"
// )

// type rocksdb struct {
// 	Path         string
// 	db           *gorocksdb.DB
// 	writeOptions *gorocksdb.WriteOptions
// 	readOptions  *gorocksdb.ReadOptions
// }

// func openRocksDB(path string) (*gorocksdb.DB, error) {
// 	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
// 	bbto.SetBlockCache(gorocksdb.NewLRUCache(64 * 1024))
// 	opts := gorocksdb.NewDefaultOptions()
// 	opts.SetBlockBasedTableFactory(bbto)
// 	// opts.setco
// 	opts.SetCreateIfMissing(true)
// 	opts.SetMaxOpenFiles(5000)
// 	opts.SetInfoLogLevel(gorocksdb.DebugInfoLogLevel)

// 	db, err := gorocksdb.OpenDb(opts, path)
// 	if err != nil {
// 		log.Error("open rocksdb error:", err)
// 		return nil, errors.New("fail to open rocksdb")
// 	}

// 	log.Infof("open rocksdb success, path:%s", path)
// 	return db, nil
// }

// func (r *rocksdb) newOptions() {
// 	var writeOptions = gorocksdb.NewDefaultWriteOptions()
// 	writeOptions.SetSync(true)

// 	var readOptions = gorocksdb.NewDefaultReadOptions()
// 	readOptions.SetFillCache(false)

// 	r.writeOptions = writeOptions
// 	r.readOptions = readOptions
// }

// func (r *rocksdb) getRocksDB(path string) (*gorocksdb.DB, error) {
// 	if r.db == nil {
// 		rdb, err := openRocksDB(path)
// 		if err != nil {
// 			log.Error("Open rocks db failed:", err)
// 			return nil, err
// 		}

// 		r.db = rdb

// 		r.newOptions()
// 	}

// 	return r.db, nil
// }

// func (r *rocksdb) Type() string {
// 	return "RocksDB"
// }

// func (r *rocksdb) Put(key string, value []byte) error {
// 	rdb, err := r.getRocksDB(r.Path)
// 	if err != nil {
// 		log.Error("Get rocks db failed:", err)
// 		return err
// 	}

// 	var k = []byte(key)
// 	return rdb.Put(r.writeOptions, k, value)
// }

// func (r *rocksdb) Get(key string) ([]byte, error) {
// 	rdb, err := r.getRocksDB(r.Path)
// 	if err != nil {
// 		log.Error("Get rocks db failed:", err)
// 		return nil, err
// 	}

// 	var k = []byte(key)
// 	slice, err := rdb.Get(r.readOptions, k)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if !slice.Exists() {
// 		return nil, datastore.ErrNotFound
// 	}

// 	defer slice.Free()

// 	return slice.Data(), nil
// }

// func (r *rocksdb) Delete(key string) error {
// 	rdb, err := r.getRocksDB(r.Path)
// 	if err != nil {
// 		log.Error("Get rocks db failed:", err)
// 		return err
// 	}

// 	var k = []byte(key)
// 	slice, err := rdb.Get(r.readOptions, k)
// 	if err != nil {
// 		return err
// 	}

// 	if !slice.Exists() {
// 		return datastore.ErrNotFound
// 	}

// 	return rdb.Delete(r.writeOptions, k)
// }

// func (r *rocksdb) GetReader(key string) (BlockReader, error) {
// 	rdb, err := r.getRocksDB(r.Path)
// 	if err != nil {
// 		log.Error("Get rocks db failed:", err)
// 		return nil, err
// 	}

// 	var k = []byte(key)
// 	slice, err := rdb.Get(r.readOptions, k)
// 	if err != nil {
// 		return nil, err
// 	}

// 	defer slice.Free()

// 	rd := bytes.NewReader(slice.Data())
// 	return &reader{rd}, nil
// }

// func (r *rocksdb) Has(key string) (exists bool, err error) {
// 	rdb, err := r.getRocksDB(r.Path)
// 	if err != nil {
// 		log.Error("Get rocks db failed:", err)
// 		return false, err
// 	}

// 	var k = []byte(key)
// 	slice, err := rdb.Get(r.readOptions, k)
// 	if err != nil {
// 		return false, err
// 	}

// 	return slice.Exists(), nil
// }

// func (r *rocksdb) GetSize(key string) (size int, err error) {
// 	rdb, err := r.getRocksDB(r.Path)
// 	if err != nil {
// 		log.Error("Get rocks db failed:", err)
// 		return 0, err
// 	}

// 	var k = []byte(key)
// 	slice, err := rdb.Get(r.readOptions, k)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return slice.Size(), nil
// }

// func (r *rocksdb) Stat() (fsutil.FsStat, error) {
// 	return fsutil.Statfs(r.Path)
// }

// func (r *rocksdb) DiskUsage() (int64, error) {
// 	si, err := fsutil.FileSize(r.Path)
// 	if err != nil {
// 		return 0, err
// 	}
// 	return si.OnDisk, nil
// }

// func (r *rocksdb) KeyCount() (int, error) {
// 	rdb, err := r.getRocksDB(r.Path)
// 	if err != nil {
// 		log.Error("Get rocks db failed:", err)
// 		return 0, err
// 	}

// 	// rocksdb.GetProperty("rocksdb.estimate-num-keys")

// 	it := rdb.NewIterator(r.readOptions)
// 	defer it.Close()

// 	it.SeekToFirst()

// 	count := 0
// 	for ; it.Valid(); it.Next() {
// 		count++
// 	}
// 	return count, nil
// }

// func (r *rocksdb) GetAllKeys() ([]string, error) {
// 	rdb, err := r.getRocksDB(r.Path)
// 	if err != nil {
// 		log.Error("Get rocks db failed:", err)
// 		return []string{}, err
// 	}

// 	// rocksdb.GetProperty("rocksdb.estimate-num-keys")

// 	it := rdb.NewIterator(r.readOptions)
// 	defer it.Close()

// 	it.SeekToFirst()

// 	keys := make([]string, 0)
// 	for ; it.Valid(); it.Next() {
// 		keys = append(keys, string(it.Key().Data()))
// 	}
// 	return keys, nil
// }

// type reader struct {
// 	r *bytes.Reader
// }

// func (r *reader) Read(p []byte) (n int, err error) {
// 	return r.r.Read(p)
// }

// func (r *reader) Close() error {
// 	return nil
// }

// func (r *reader) Seek(offset int64, whence int) (int64, error) {
// 	return r.r.Seek(offset, whence)
// }

// func (r *reader) Size() int64 {
// 	return r.r.Size()
// }
