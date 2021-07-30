package badgerdbprovider

import (
	"bytes"

	"github.com/dgraph-io/badger"
)

type Iterator struct {
	db    *badger.DB
	biter *badger.Iterator
	opts  *badger.IteratorOptions
	txn   *badger.Txn
	start []byte
	end   []byte
	first bool
}

func NewIterator(db *badger.DB, opts badger.IteratorOptions, start []byte, end []byte) *Iterator {
	txn := db.NewTransaction(false)
	iter := &Iterator{
		db:    db,
		biter: txn.NewIterator(opts),
		start: start,
		end:   end,
		txn:   txn,
		opts:  &opts,
		first: true,
	}

	if iter.isRangeIter() {
		iter.biter.Seek(iter.start)
	} else {
		iter.biter.Rewind()
	}

	return iter
}

func (iter *Iterator) Key() []byte {
	if iter.biter.Valid() {
		return iter.biter.Item().Key()
	}
	return nil
}

func (iter *Iterator) Value() []byte {
	if iter.biter.Valid() {
		item := iter.biter.Item()
		value, err := item.ValueCopy(nil)
		if err != nil {
			return nil
		}
		return value
	}

	return nil
}

func (iter *Iterator) Next() bool {
	if !iter.isValid() {
		return false
	}

	if iter.first {
		iter.first = false
		return true
	}

	iter.biter.Next()
	if iter.isRangeIter() {
		// if start and end are both not nil, this is a range iter
		if iter.biter.Valid() {
			item := iter.biter.Item()
			return bytes.Compare(item.Key(), iter.start) >= 0 && bytes.Compare(item.Key(), iter.end) < 0
		}
		return false
	}
	//else should be prefix iter
	return iter.isValid()
}

func (iter *Iterator) First() bool {
	if iter.isRangeIter() {
		iter.biter.Seek(iter.start)
	} else {
		iter.biter.Rewind()
	}

	return iter.isValid()
}

func (iter *Iterator) Error() error {
	return nil
}

func (iter *Iterator) Release() {
	iter.biter.Close()
	iter.txn.Discard()
}

func (iter *Iterator) isValid() bool {
	if iter.biter.Valid() {
		if len(iter.opts.Prefix) != 0 {
			return iter.biter.ValidForPrefix(iter.opts.Prefix)
		}
		return true
	}

	return false
}

func (iter *Iterator) isRangeIter() bool {
	return len(iter.start) != 0 && len(iter.end) != 0
}
