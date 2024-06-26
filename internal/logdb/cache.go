// Copyright 2017-2019 Lei Ni (nilei81@gmail.com) and other contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logdb

import (
	"sync"

	"github.com/foreeest/dragonboat/raftio"
	pb "github.com/foreeest/dragonboat/raftpb"
)

type cache struct {
	nodeInfo       map[raftio.NodeInfo]struct{}
	ps             map[raftio.NodeInfo]pb.State
	lastEntryBatch map[raftio.NodeInfo]pb.EntryBatch
	maxIndex       map[raftio.NodeInfo]uint64
	snapshotIndex  map[raftio.NodeInfo]uint64
	mu             sync.Mutex
}

func newCache() *cache {
	return &cache{
		nodeInfo:       make(map[raftio.NodeInfo]struct{}),
		ps:             make(map[raftio.NodeInfo]pb.State),
		lastEntryBatch: make(map[raftio.NodeInfo]pb.EntryBatch),
		maxIndex:       make(map[raftio.NodeInfo]uint64),
		snapshotIndex:  make(map[raftio.NodeInfo]uint64),
	}
}

func (r *cache) setNodeInfo(shardID uint64, replicaID uint64) bool {
	key := raftio.NodeInfo{ShardID: shardID, ReplicaID: replicaID}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.nodeInfo[key]
	if !ok {
		r.nodeInfo[key] = struct{}{}
	}
	return !ok
}

func (r *cache) setState(shardID uint64, replicaID uint64, st pb.State) bool {
	key := raftio.NodeInfo{ShardID: shardID, ReplicaID: replicaID}
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.ps[key]
	if !ok {
		r.ps[key] = st
		return true
	}
	if pb.IsStateEqual(v, st) {
		return false
	}
	r.ps[key] = st
	return true
}

func (r *cache) setSnapshotIndex(shardID uint64, replicaID uint64, index uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := raftio.NodeInfo{ShardID: shardID, ReplicaID: replicaID}
	r.snapshotIndex[key] = index
}

func (r *cache) trySaveSnapshot(shardID uint64,
	replicaID uint64, index uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := raftio.NodeInfo{ShardID: shardID, ReplicaID: replicaID}
	v, ok := r.snapshotIndex[key]
	if !ok {
		r.snapshotIndex[key] = index
		return true
	}
	return index > v
}

func (r *cache) setMaxIndex(shardID uint64, replicaID uint64, maxIndex uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := raftio.NodeInfo{ShardID: shardID, ReplicaID: replicaID}
	r.maxIndex[key] = maxIndex
}

func (r *cache) getMaxIndex(shardID uint64, replicaID uint64) (uint64, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := raftio.NodeInfo{ShardID: shardID, ReplicaID: replicaID}
	v, ok := r.maxIndex[key]
	if !ok {
		return 0, false
	}
	return v, true
}

func (r *cache) setLastBatch(shardID uint64,
	replicaID uint64, eb pb.EntryBatch) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := raftio.NodeInfo{ShardID: shardID, ReplicaID: replicaID}
	oeb, ok := r.lastEntryBatch[key]
	if !ok {
		oeb = pb.EntryBatch{Entries: make([]pb.Entry, 0, len(eb.Entries))}
	} else {
		oeb.Entries = oeb.Entries[:0]
	}
	oeb.Entries = append(oeb.Entries, eb.Entries...)
	r.lastEntryBatch[key] = oeb
}

func (r *cache) getLastBatch(shardID uint64,
	replicaID uint64, lb pb.EntryBatch) (pb.EntryBatch, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := raftio.NodeInfo{ShardID: shardID, ReplicaID: replicaID}
	v, ok := r.lastEntryBatch[key]
	if !ok {
		return v, false
	}
	lb.Entries = lb.Entries[:0]
	lb.Entries = append(lb.Entries, v.Entries...)
	return lb, true
}
