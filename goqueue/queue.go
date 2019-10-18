package goqueue

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	StoreBucket = []byte("stores")
	IdsBucket   = []byte("ids")
	StatBucket  = []byte("stats")
)

type Stats struct {
	CurrentID int       `json:"current_id"`
	Size      int       `json:"size"`
	ReadSize  int       `json:"read_size"`
	ReplySize int       `json:"reply_size"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (this *Stats) String() string {
	return fmt.Sprintf("size: %d, read size: %d, reply size: %d, created: %s, updated: %s", this.Size, this.ReadSize, this.ReplySize, this.CreatedAt.String(), this.UpdatedAt.String())
}

const (
	StatusPending    = iota // 0
	StatusProcessing        // 1
	StatusInvalid           // 2
	StatusOK                // 3
)

type Item struct {
	ID        int       `json:"id"`
	Message   string    `json:"message"`
	Status    int       `json:"status"`
	Error     string    `json:"error"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewItem(msg string) *Item {
	return &Item{
		ID:        0,
		Message:   msg,
		Status:    StatusPending,
		Error:     "",
		CreatedAt: time.Now(),
	}
}

func NewItemFromBytes(data []byte) (*Item, error) {
	item := &Item{}
	err := json.Unmarshal(data, item)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (this *Item) Bytes() ([]byte, error) {
	return json.Marshal(this)
}

func (this *Item) Key() []byte {
	return itob(this.ID)
}

type Queue struct {
	name     string
	dataPath string
	db       *bolt.DB
	stats    *Stats
}

func New(name, dataPath string) (*Queue, error) {
	this := &Queue{
		name:     name,
		dataPath: dataPath,
	}

	this.stats = &Stats{
		CurrentID: 0,
		Size:      0,
		ReadSize:  0,
		ReplySize: 0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := this.initDB(); err != nil {
		return nil, err
	}

	return this, nil
}

func (this *Queue) initDB() error {
	var err error
	dbfile := filepath.Join(this.dataPath, this.name)
	this.db, err = bolt.Open(dbfile, 0600, nil)
	if err != nil {
		return err
	}

	return this.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(StoreBucket)
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists(IdsBucket)
		if err != nil {
			return err
		}

		_, err = tx.CreateBucket(StatBucket)
		if err == bolt.ErrBucketExists {
			if err := this.getStats(tx); err != nil {
				return err
			}
		} else {
			if err := this.saveStats(tx); err != nil {
				return err
			}
		}

		return nil
	})
}

func (this *Queue) Get() (string, error) {
	if this.db == nil {
		return "", nil
	}

	var msg string
	if err := this.db.Update(func(tx *bolt.Tx) error {
		storeBucket := tx.Bucket(StoreBucket)
		if this.stats.ReadSize >= storeBucket.Stats().KeyN {
			return nil
		}

		var data []byte
		cursor := storeBucket.Cursor()
		if this.stats.ReadSize == 0 {
			_, data = cursor.First()
		} else {
			cursor.Seek(itob(this.stats.CurrentID))
			_, data = cursor.Next()
		}
		if data == nil {
			return nil
		}
		item, err := NewItemFromBytes(cloneBytes(data))
		if err != nil {
			return err
		}

		msg = item.Message

		// 修改状态
		item.Status = StatusProcessing
		if err := this.putItem(tx, item); err != nil {
			return err
		}

		this.stats.CurrentID = item.ID
		this.stats.ReadSize++
		if err := this.saveStats(tx); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return "", err
	}

	return msg, nil
}

func (this *Queue) Put(msg string) error {
	if this.db == nil {
		return nil
	}

	if this.Exists(msg) {
		return nil
	}

	item := NewItem(msg)
	return this.db.Update(func(tx *bolt.Tx) error {
		if err := this.putItem(tx, item); err != nil {
			return err
		}

		this.stats.Size++
		return this.saveStats(tx)
	})
}

func (this *Queue) Find(offset, limit int) []*Item {
	var items []*Item
	if offset <= 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10000
	}
	if offset > this.stats.Size {
		return items
	}
	if limit > 100000 {
		limit = 100000
	}
	this.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(StoreBucket)
		if b == nil {
			return nil
		}
		c := b.Cursor()
		if c == nil {
			return nil
		}

		i := 0
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if i >= offset {
				item, _ := NewItemFromBytes(cloneBytes(v))
				items = append(items, item)
			}
			if len(items) >= limit {
				return nil
			}
			i++
		}

		return nil
	})

	return items
}

// 返回队列中剩余数量
func (this *Queue) Size() int {
	lag := this.stats.Size - this.stats.ReadSize
	if lag <= 0 {
		lag = 0
	}

	return lag
	// var size int

	// this.db.View(func(tx *bolt.Tx) error {
	// 	b := tx.Bucket(StoreBucket)
	// 	if b == nil {
	// 		return nil
	// 	}
	// 	size = b.Stats().KeyN
	// 	return nil
	// })

	// return size
}

func (this *Queue) Exists(msg string) bool {
	if this.db == nil {
		return false
	}

	var ret bool
	this.db.View(func(tx *bolt.Tx) error {
		ret = this.visited(tx, msg)

		return nil
	})

	return ret
}

func (this *Queue) Reply(msg string, status int, errMsg string) error {
	if !this.Exists(msg) {
		return nil
	}

	if status != StatusOK {
		status = StatusInvalid
	}
	item := NewItem(msg)
	item.Status = status
	item.Error = errMsg
	item.UpdatedAt = time.Now()
	return this.db.Update(func(tx *bolt.Tx) error {
		if err := this.putItem(tx, item); err != nil {
			return err
		}

		this.stats.ReplySize++
		return this.saveStats(tx)
	})
}

func (this *Queue) ReplyOK(msg string) error {
	return this.Reply(msg, StatusOK, "")
}

func (this *Queue) ReplyInvalid(msg, errMsg string) error {
	return this.Reply(msg, StatusInvalid, errMsg)
}

func (this *Queue) Stats() *Stats {
	return this.stats
}

func (this *Queue) Close() error {
	return this.db.Close()
}

func (this *Queue) getStats(tx *bolt.Tx) error {
	if tx == nil {
		return nil
	}

	statBucket := tx.Bucket(StatBucket)
	data := statBucket.Get(StatBucket)

	if err := json.Unmarshal(cloneBytes(data), this.stats); err != nil {
		return err
	}

	return nil
}

func (this *Queue) saveStats(tx *bolt.Tx) error {
	if tx == nil {
		return nil
	}

	statBucket := tx.Bucket(StatBucket)
	this.stats.UpdatedAt = time.Now()
	data, err := json.Marshal(this.stats)
	if err != nil {
		return err
	}

	if err := statBucket.Put(StatBucket, cloneBytes(data)); err != nil {
		return err
	}

	return nil
}

func (this *Queue) putItem(tx *bolt.Tx, item *Item) error {
	if tx == nil {
		return nil
	}

	var isAdd bool
	id, err := this.getID(tx, item.Message)
	if err != nil {
		return err
	}
	item.ID = id

	storeBucket := tx.Bucket(StoreBucket)
	if item.ID == 0 {
		isAdd = true
		id, err := storeBucket.NextSequence()
		if err != nil {
			return err
		}
		item.ID = int(id)
	}

	data, err := item.Bytes()
	if err != nil {
		return err
	}

	if err := storeBucket.Put(itob(item.ID), cloneBytes(data)); err != nil {
		return err
	}

	if isAdd {
		if err := this.storeID(tx, item.Message, item.ID); err != nil {
			return err
		}
	}

	return nil
}

func (this *Queue) storeID(tx *bolt.Tx, msg string, id int) error {
	buck := tx.Bucket(IdsBucket)
	if buck == nil {
		return nil
	}

	hBucket, err := buck.CreateBucketIfNotExists(getIdsBucket(msg))
	if err != nil {
		return err
	}

	return hBucket.Put([]byte(msg), itob(id))
}

func (this *Queue) visited(tx *bolt.Tx, msg string) bool {
	id, _ := this.getID(tx, msg)
	return id > 0
}

func (this *Queue) getID(tx *bolt.Tx, msg string) (int, error) {
	bucket := tx.Bucket(IdsBucket)
	if bucket == nil {
		return 0, nil
	}

	b := bucket.Bucket(getIdsBucket(msg))
	if b == nil {
		return 0, nil
	}
	v := b.Get([]byte(msg))
	if v == nil {
		return 0, nil
	}

	return btoi(cloneBytes(v))
}

func getIdsBucket(str string) []byte {
	h := md5.New()
	io.WriteString(h, str)
	v := hex.EncodeToString(h.Sum(nil))
	return []byte(v[:2])
}

func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func btoi(v []byte) (int, error) {
	var id int64
	r := bytes.NewReader(v)
	err := binary.Read(r, binary.BigEndian, &id)
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func cloneBytes(v []byte) []byte {
	var clone = make([]byte, len(v))
	copy(clone, v)
	return clone
}
