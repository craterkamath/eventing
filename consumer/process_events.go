package consumer

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/couchbase/eventing/common"
	"github.com/couchbase/eventing/util"
	sc "github.com/couchbase/indexing/secondary/common"
	"github.com/couchbase/indexing/secondary/dcp"
	mcd "github.com/couchbase/indexing/secondary/dcp/transport"
	"github.com/couchbase/indexing/secondary/logging"
	"github.com/couchbase/nitro/plasma"
)

func (c *Consumer) doDCPEventProcess() {
	for {
		select {
		case e, ok := <-c.aggDCPFeed:
			if ok == false {
				logging.Infof("CRDP[%s:%s:%s:%d] Closing DCP feed for bucket %q",
					c.app.AppName, c.workerName, c.tcpPort, c.Pid(), c.bucket)

				c.stopCheckpointingCh <- true
				c.producer.CleanupDeadConsumer(c)
				return
			}

			if _, ok := c.dcpMessagesProcessed[e.Opcode]; !ok {
				c.dcpMessagesProcessed[e.Opcode] = 0
			}
			c.dcpMessagesProcessed[e.Opcode]++

			switch e.Opcode {
			case mcd.DCP_MUTATION:
				switch e.Datatype {
				case dcpDatatypeJSON:
					c.sendDcpEvent(e)
				case dcpDatatypeJSONXattr:
					xattrLen := binary.BigEndian.Uint32(e.Value[0:])
					xattrData := e.Value[4 : 4+xattrLen-1]

					xIndex := bytes.Index(xattrData, []byte(xattrPrefix))
					xattrVal := xattrData[xIndex+len(xattrPrefix)+1:]

					var xMeta xattrMetadata
					err := json.Unmarshal(xattrVal, &xMeta)
					if err != nil {
						logging.Errorf("CRDP[%s:%s:%s:%d] Key: %v xattrVal: %v",
							c.app.AppName, c.workerName, c.tcpPort, c.Pid(), string(e.Key), string(xattrVal))
						continue
					}

					cas, err := util.ConvertBigEndianToUint64([]byte(xMeta.Cas))
					if err != nil {
						logging.Errorf("CRDP[%s:%s:%s:%d] Key: %v Failed to convert cas string from kv to uint64, err: %v",
							c.app.AppName, c.workerName, c.tcpPort, c.Pid(), string(e.Key), err)
						continue
					}
					// Send mutation to V8 CPP worker _only_ when DcpEvent.Cas != Cas field in xattr
					if cas != e.Cas {
						e.Value = e.Value[4+xattrLen:]
						c.sendDcpEvent(e)
					} else {
						logging.Debugf("CRPO[%s:%s:%s:%d] Skipping recursive mutation for Key: %v vb: %v, xmeta: %#v",
							c.app.AppName, c.workerName, c.tcpPort, c.Pid(), string(e.Key), e.VBucket, xMeta)

						c.storeTimerEvent(e.VBucket, e.Seqno, e.Expiry, string(e.Key), &xMeta)
					}
				}

			case mcd.DCP_DELETION:
				c.sendDcpEvent(e)

			case mcd.DCP_STREAMREQ:

				logging.Infof("CRDP[%s:%s:%s:%d] vb: %d status: %v",
					c.app.AppName, c.workerName, c.tcpPort, c.Pid(), e.VBucket, e.Status)

				if e.Status == mcd.SUCCESS {

					c.vbProcessingStats.updateVbStat(e.VBucket, "assigned_worker", c.ConsumerName())
					c.vbProcessingStats.updateVbStat(e.VBucket, "current_vb_owner", c.HostPortAddr())
					c.vbProcessingStats.updateVbStat(e.VBucket, "dcp_stream_status", dcpStreamRunning)
					c.vbProcessingStats.updateVbStat(e.VBucket, "last_processed_seq_no", uint64(0))
					c.vbProcessingStats.updateVbStat(e.VBucket, "node_uuid", c.uuid)

					vbFlog := &vbFlogEntry{streamReqRetry: false, statusCode: e.Status}

					var vbBlob vbucketKVBlob
					var cas uint64

					vbKey := fmt.Sprintf("%s_vb_%s", c.app.AppName, strconv.Itoa(int(e.VBucket)))

					util.Retry(util.NewFixedBackoff(bucketOpRetryInterval), getOpCallback, c, vbKey, &vbBlob, &cas, false)

					vbuuid, seqNo, err := e.FailoverLog.Latest()
					if err != nil {
						logging.Errorf("CRDP[%s:%s:%s:%d] Failure to get latest failover log vb: %d err: %v, not updating metadata",
							c.app.AppName, c.workerName, c.tcpPort, c.Pid(), e.VBucket, err)
						c.vbFlogChan <- vbFlog
						continue
					}

					// Update metadata with latest vbuuid and rolback seq no.
					vbBlob.LastSeqNoProcessed = seqNo
					vbBlob.VBuuid = vbuuid
					vbBlob.AssignedWorker = c.ConsumerName()
					vbBlob.CurrentVBOwner = c.HostPortAddr()
					vbBlob.DCPStreamStatus = dcpStreamRunning
					vbBlob.LastCheckpointTime = time.Now().Format(time.RFC3339)

					entry := OwnershipEntry{
						AssignedWorker: c.ConsumerName(),
						CurrentVBOwner: c.HostPortAddr(),
						Operation:      dcpStreamRunning,
						Timestamp:      time.Now().String(),
					}
					vbBlob.OwnershipHistory = append(vbBlob.OwnershipHistory, entry)

					util.Retry(util.NewFixedBackoff(bucketOpRetryInterval), casOpCallback, c, vbKey, &vbBlob, &cas)

					c.vbFlogChan <- vbFlog
					continue
				}

				if e.Status == mcd.KEY_EEXISTS || e.Status == mcd.NOT_MY_VBUCKET {
					vbFlog := &vbFlogEntry{streamReqRetry: false, statusCode: e.Status}
					c.vbFlogChan <- vbFlog
					continue
				}

				if e.Status == mcd.EINVAL || e.Status == mcd.ROLLBACK || e.Status == mcd.ENOMEM {
					vbFlog := &vbFlogEntry{
						seqNo:          e.Seqno,
						streamReqRetry: true,
						statusCode:     e.Status,
						vb:             e.VBucket,
						flog:           e.FailoverLog,
					}
					c.vbFlogChan <- vbFlog
				}
			case mcd.DCP_STREAMEND:
				// Cleanup entry for vb for which stream_end has been received from vbPorcessingStats
				// which will allow vbTakeOver background routine to start up new stream from
				// new KV node, where the vbucket has been migrated

				logging.Infof("CRVT[%s:%s:%s:%d] vb: %v, got STREAMEND", c.app.AppName, c.workerName, c.tcpPort, c.Pid(), e.VBucket)

				//Store the latest state of vbucket processing stats in the metadata bucket
				vbKey := fmt.Sprintf("%s_vb_%s", c.app.AppName, strconv.Itoa(int(e.VBucket)))
				var vbBlob vbucketKVBlob
				var cas uint64

				util.Retry(util.NewFixedBackoff(bucketOpRetryInterval), getOpCallback, c, vbKey, &vbBlob, &cas, false)

				vbBlob.AssignedWorker = ""
				vbBlob.CurrentVBOwner = ""
				vbBlob.DCPStreamStatus = dcpStreamStopped
				vbBlob.PreviousAssignedWorker = c.ConsumerName()
				vbBlob.PreviousEventingDir = c.eventingDir
				vbBlob.PreviousNodeUUID = c.NodeUUID()
				vbBlob.PreviousVBOwner = c.HostPortAddr()

				vbBlob.LastSeqNoProcessed = c.vbProcessingStats.getVbStat(e.VBucket, "last_processed_seq_no").(uint64)
				entry := OwnershipEntry{
					AssignedWorker: c.ConsumerName(),
					CurrentVBOwner: c.HostPortAddr(),
					Operation:      dcpStreamStopped,
					Timestamp:      time.Now().String(),
				}
				vbBlob.OwnershipHistory = append(vbBlob.OwnershipHistory, entry)

				util.Retry(util.NewFixedBackoff(bucketOpRetryInterval), casOpCallback, c, vbKey, &vbBlob, &cas)

				c.vbProcessingStats.updateVbStat(e.VBucket, "assigned_worker", "")
				c.vbProcessingStats.updateVbStat(e.VBucket, "current_vb_owner", "")
				c.vbProcessingStats.updateVbStat(e.VBucket, "dcp_stream_status", "stopped")
				c.vbProcessingStats.updateVbStat(e.VBucket, "node_uuid", "")

				c.Lock()
				// Check if vbucket related entry already exists, if yes - then clean it up
				// and close all associated FDs
				if _, ok := c.vbPlasmaStoreMap[e.VBucket]; ok {
					c.signalProcessTimerPlasmaCloseCh <- e.VBucket
					<-c.signalProcessTimerPlasmaCloseAckCh

					// Instead of sending message over channel - to clean up plasma.Writer
					// instances who are responsible for storing timers events into
					// plasma store, cleaning up vbucket specific plasma.Writer instances
					// directly. Reason being, c.signalStoreTimerPlasmaCloseCh and
					// c.signalStoreTimerPlasmaCloseAckCh are being listened to/written to
					// on current control path within the select statement
					_, ok := c.vbPlasmaWriter[e.VBucket]
					if ok {
						delete(c.vbPlasmaWriter, e.VBucket)
					}

					c.closePlasmaHandle(e.VBucket)
				}

				c.Unlock()

				if c.checkIfCurrentConsumerShouldOwnVb(e.VBucket) {
					logging.Infof("CRVT[%s:%s:%s:%d] vb: %v, got STREAMEND, needs to be reclaimed",
						c.app.AppName, c.workerName, c.tcpPort, c.Pid(), e.VBucket)
					c.Lock()
					c.vbsRemainingToRestream = append(c.vbsRemainingToRestream, e.VBucket)
					c.Unlock()
				}

			default:
			}

		case e, ok := <-c.timerEntryCh:
			if ok == false {
				logging.Infof("CRDP[%s:%s:%s:%d] Closing timer chan", c.app.AppName, c.workerName, c.tcpPort, c.Pid())

				c.stopCheckpointingCh <- true
				c.producer.CleanupDeadConsumer(c)
				return
			}

			c.timerMessagesProcessed++
			c.sendTimerEvent(e)

		case <-c.statsTicker.C:

			util.Retry(util.NewFixedBackoff(clusterOpRetryInterval), getEventingNodeAddrOpCallback, c)
			vbsOwned := c.getCurrentlyOwnedVbs()
			if len(vbsOwned) > 0 {
				c.RLock()

				countMsg, dcpOpCount, tStamp := util.SprintDCPCounts(c.dcpMessagesProcessed)

				diff := tStamp.Sub(c.opsTimestamp)
				opsDiff := dcpOpCount - c.dcpOpsProcessed
				seconds := int(diff.Nanoseconds() / (1000 * 1000 * 1000))
				if seconds > 0 {
					c.dcpOpsProcessedPSec = int(opsDiff) / seconds
				}

				logging.Infof("CRDP[%s:%s:%s:%d] DCP events processed: %s V8 events processed: %s Timer events processed: %v, vbs owned len: %d vbs owned:[%d..%d]",
					c.app.AppName, c.workerName, c.tcpPort, c.Pid(), countMsg, util.SprintV8Counts(c.v8WorkerMessagesProcessed),
					c.timerMessagesProcessed, len(c.getCurrentlyOwnedVbs()), vbsOwned[0], vbsOwned[len(vbsOwned)-1])

				c.opsTimestamp = tStamp
				c.dcpOpsProcessed = dcpOpCount
				c.RUnlock()
			}

		case vb := <-c.signalStoreTimerPlasmaCloseCh:
			// Rebalance takeover routine will send signal on this channel to signify
			// stopping of any plasma.Writer instance for a specific vbucket
			_, ok := c.vbPlasmaWriter[vb]
			if ok {
				delete(c.vbPlasmaWriter, vb)
			}

			// sends ack message back to rebalance takeover routine, so that it could
			// safely call Close() on vb specific plasma store
			c.signalStoreTimerPlasmaCloseAckCh <- vb

		case <-c.stopConsumerCh:

			logging.Errorf("CRDP[%s:%s:%s:%d] Socket belonging to V8 consumer died",
				c.app.AppName, c.workerName, c.tcpPort, c.Pid())
			c.stopCheckpointingCh <- true
			c.producer.CleanupDeadConsumer(c)
			return

		case <-c.gracefulShutdownChan:
			return
		}
	}
}

func (c *Consumer) startDcp(dcpConfig map[string]interface{}, flogs couchbase.FailoverLog) {

	logging.Infof("CRDP[%s:%s:%s:%d] no. of vbs owned: %d",
		c.app.AppName, c.workerName, c.tcpPort, c.Pid(), len(c.vbnos))

	util.Retry(util.NewFixedBackoff(clusterOpRetryInterval), getEventingNodeAddrOpCallback, c)

	vbSeqnos, err := sc.BucketSeqnos(c.producer.NsServerHostPort(), "default", c.bucket)
	if err != nil && c.dcpStreamBoundary != common.DcpEverything {
		logging.Errorf("CRDP[%s:%s:%s:%d] Failed to fetch vb seqnos, err: %v", c.app.AppName, c.workerName, c.tcpPort, c.Pid(), err)
		return
	}

	logging.Infof("CRDP[%s:%s:%s:%d] get_all_vb_seqnos: len => %d dump => %v",
		c.app.AppName, c.workerName, c.tcpPort, c.Pid(), len(vbSeqnos), vbSeqnos)

	for vbno, flog := range flogs {
		vbuuid, _, _ := flog.Latest()

		vbKey := fmt.Sprintf("%s_vb_%s", c.app.AppName, strconv.Itoa(int(vbno)))
		var vbBlob vbucketKVBlob
		var cas, start uint64
		var isNoEnt bool

		util.Retry(util.NewFixedBackoff(bucketOpRetryInterval), getOpCallback, c, vbKey, &vbBlob, &cas, true, &isNoEnt)
		if isNoEnt {

			// Storing vbuuid in metadata bucket, will be required for start
			// stream later on
			vbBlob.VBuuid = vbuuid
			vbBlob.VBId = vbno
			vbBlob.AssignedWorker = c.ConsumerName()
			vbBlob.CurrentVBOwner = c.HostPortAddr()

			// Assigning previous owner and worker to current consumer
			vbBlob.PreviousAssignedWorker = c.ConsumerName()
			vbBlob.PreviousNodeUUID = c.NodeUUID()
			vbBlob.PreviousVBOwner = c.HostPortAddr()
			vbBlob.PreviousEventingDir = c.eventingDir

			util.Retry(util.NewFixedBackoff(bucketOpRetryInterval), setOpCallback, c, vbKey, &vbBlob)

			switch c.dcpStreamBoundary {
			case common.DcpEverything:
				start = uint64(0)
				c.dcpRequestStreamHandle(vbno, &vbBlob, start)
			case common.DcpFromNow:
				start = uint64(vbSeqnos[int(vbno)])
				c.dcpRequestStreamHandle(vbno, &vbBlob, start)
			}
		} else {

			if vbBlob.NodeUUID == c.NodeUUID() {
				start = vbBlob.LastSeqNoProcessed

				c.dcpRequestStreamHandle(vbno, &vbBlob, start)
			}
		}
	}
}

func (c *Consumer) addToAggChan(dcpFeed *couchbase.DcpFeed, cancelCh <-chan bool) {
	go func(dcpFeed *couchbase.DcpFeed) {
		defer func() {
			if r := recover(); r != nil {
				logging.Errorf("CRDP[%s:%s:%s:%d] addToAggChan: panic and recover, %v", c.app.AppName, c.workerName, c.tcpPort, c.Pid(), r)
			}
		}()

		for {
			select {
			case e, ok := <-dcpFeed.C:
				if ok == false {
					var kvAddr string
					for addr, feed := range c.kvHostDcpFeedMap {
						if feed == dcpFeed {
							kvAddr = addr
						}
					}

					logging.Infof("CRDP[%s:%s:%s:%d] Closing dcp feed: %v for bucket: %s",
						c.app.AppName, c.workerName, c.tcpPort, c.Pid(), dcpFeed.DcpFeedName(), c.bucket)
					c.Lock()
					delete(c.kvHostDcpFeedMap, kvAddr)
					c.Unlock()

					return
				}

				if e.Opcode == mcd.DCP_STREAMEND || e.Opcode == mcd.DCP_STREAMREQ {
					logging.Infof("CRDP[%s:%s:%s:%d] addToAggChan dcpFeed name: %v vb: %v Opcode: %v Status: %v",
						c.app.AppName, c.workerName, c.tcpPort, c.Pid(), dcpFeed.DcpFeedName(), e.VBucket, e.Opcode, e.Status)
				}

				c.aggDCPFeed <- e

			case <-cancelCh:
				return
			}
		}
	}(dcpFeed)
}

func (c *Consumer) cleanupStaleDcpFeedHandles() {
	kvAddrsPerVbMap := make(map[string]struct{})
	for _, kvAddr := range c.kvVbMap {
		kvAddrsPerVbMap[kvAddr] = struct{}{}
	}

	var kvAddrListPerVbMap []string
	for kvAddr := range kvAddrsPerVbMap {
		kvAddrListPerVbMap = append(kvAddrListPerVbMap, kvAddr)
	}

	var kvHostDcpFeedMapEntries []string
	c.RLock()
	for kvAddr := range c.kvHostDcpFeedMap {
		kvHostDcpFeedMapEntries = append(kvHostDcpFeedMapEntries, kvAddr)
	}
	c.RUnlock()

	kvAddrDcpFeedsToClose := util.SliceDifferences(kvHostDcpFeedMapEntries, kvAddrListPerVbMap)

	if len(kvAddrDcpFeedsToClose) > 0 {
		util.Retry(util.NewFixedBackoff(bucketOpRetryInterval), populateDcpFeedVbEntriesCallback, c)
	}

	for _, kvAddr := range kvAddrDcpFeedsToClose {
		c.kvHostDcpFeedMap[kvAddr].Close()

		c.Lock()
		vbsMetadataToUpdate := c.dcpFeedVbMap[c.kvHostDcpFeedMap[kvAddr]]
		delete(c.kvHostDcpFeedMap, kvAddr)
		c.Unlock()

		for _, vbno := range vbsMetadataToUpdate {
			c.clearUpOnwershipInfoFromMeta(vbno)
		}
	}
}

func (c *Consumer) clearUpOnwershipInfoFromMeta(vbno uint16) {
	var vbBlob vbucketKVBlob
	var cas uint64
	vbKey := fmt.Sprintf("%s_vb_%s", c.app.AppName, strconv.Itoa(int(vbno)))
	util.Retry(util.NewFixedBackoff(bucketOpRetryInterval), getOpCallback, c, vbKey, &vbBlob, &cas, false)

	vbBlob.AssignedWorker = ""
	vbBlob.CurrentVBOwner = ""
	vbBlob.DCPStreamStatus = dcpStreamStopped
	vbBlob.LastCheckpointTime = time.Now().Format(time.RFC3339)
	vbBlob.LastSeqNoProcessed = c.vbProcessingStats.getVbStat(vbno, "last_processed_seq_no").(uint64)
	vbBlob.NodeUUID = ""
	vbBlob.PreviousAssignedWorker = c.ConsumerName()
	vbBlob.PreviousEventingDir = c.eventingDir
	vbBlob.PreviousNodeUUID = c.NodeUUID()
	vbBlob.PreviousVBOwner = c.HostPortAddr()

	c.vbProcessingStats.updateVbStat(vbno, "assigned_worker", vbBlob.AssignedWorker)
	c.vbProcessingStats.updateVbStat(vbno, "current_vb_owner", vbBlob.CurrentVBOwner)
	c.vbProcessingStats.updateVbStat(vbno, "dcp_stream_status", vbBlob.DCPStreamStatus)
	c.vbProcessingStats.updateVbStat(vbno, "node_uuid", vbBlob.NodeUUID)

	util.Retry(util.NewFixedBackoff(bucketOpRetryInterval), casOpCallback, c, vbKey, &vbBlob, &cas)
}

func (c *Consumer) dcpRequestStreamHandle(vbno uint16, vbBlob *vbucketKVBlob, start uint64) error {

	c.cbBucket.Refresh()

	util.Retry(util.NewFixedBackoff(clusterOpRetryInterval), getKvVbMap, c)
	vbKvAddr := c.kvVbMap[vbno]

	// Closing feeds for KV hosts which are no more present in kv vb map
	c.cleanupStaleDcpFeedHandles()

	dcpFeed, ok := c.kvHostDcpFeedMap[vbKvAddr]
	if !ok {
		feedName := couchbase.DcpFeedName("eventing:" + c.HostPortAddr() + "_" + vbKvAddr + "_" + c.workerName)
		util.Retry(util.NewFixedBackoff(bucketOpRetryInterval), startDCPFeedOpCallback, c, feedName, dcpConfig, vbKvAddr)
		dcpFeed = c.kvHostDcpFeedMap[vbKvAddr]

		cancelCh := make(chan bool, 1)
		c.dcpFeedCancelChs = append(c.dcpFeedCancelChs, cancelCh)
		c.addToAggChan(dcpFeed, cancelCh)

		logging.Infof("CRDP[%s:%s:%s:%d] vb: %d kvAddr: %v Started up new dcpFeed",
			c.app.AppName, c.workerName, c.tcpPort, c.Pid(), vbno, vbKvAddr)
	}

	c.vbDcpFeedMap[vbno] = dcpFeed

	opaque, flags := uint16(vbno), uint32(0)
	end := uint64(0xFFFFFFFFFFFFFFFF)

	snapStart, snapEnd := start, start

	logging.Infof("CRDP[%s:%s:%s:%d] vb: %d DCP stream start vbKvAddr: %v vbuuid: %d startSeq: %d snapshotStart: %d snapshotEnd: %d",
		c.app.AppName, c.workerName, c.tcpPort, c.Pid(), vbno, vbKvAddr, vbBlob.VBuuid, start, snapStart, snapEnd)

	err := dcpFeed.DcpRequestStream(vbno, opaque, flags, vbBlob.VBuuid, start, end, snapStart, snapEnd)
	if err != nil {
		logging.Errorf("CRDP[%s:%s:%s:%d] vb: %d STREAMREQ call failed on dcpFeed: %v, err: %v",
			c.app.AppName, c.workerName, c.tcpPort, c.Pid(), vbno, dcpFeed.DcpFeedName(), err)
		if c.checkIfCurrentConsumerShouldOwnVb(vbno) {
			c.Lock()
			c.vbsRemainingToRestream = append(c.vbsRemainingToRestream, vbno)
			c.Unlock()

			c.clearUpOnwershipInfoFromMeta(vbno)
		}
		return err
	}

loop:
	vbFlog := <-c.vbFlogChan

	if !vbFlog.streamReqRetry && vbFlog.statusCode == mcd.SUCCESS {
		logging.Infof("CRDP[%s:%s:%s:%d] vb: %d DCP Stream created", c.app.AppName, c.workerName, c.tcpPort, c.Pid(), vbno)

		vbPlasmaDir := fmt.Sprintf("%v/%v/%v_timer.data", c.eventingDir, c.app.AppName, vbno)

		cfg := plasma.DefaultConfig()
		cfg.File = vbPlasmaDir
		cfg.AutoLSSCleaning = autoLssCleaning
		cfg.MaxDeltaChainLen = maxDeltaChainLen
		cfg.MaxPageItems = maxPageItems
		cfg.MinPageItems = minPageItems
		cfg.UseMemoryMgmt = useMemManagement

		c.vbPlasmaStoreMap[vbno], err = plasma.New(cfg)
		if err != nil {
			logging.Infof("CRDP[%s:%s:%s:%d] vb: %v Failed to create plasma store instance, err: %v",
				c.app.AppName, c.workerName, c.tcpPort, c.Pid(), vbno, err)
			return err
		}

		c.vbPlasmaReader[vbno] = c.vbPlasmaStoreMap[vbno].NewWriter()
		c.vbPlasmaWriter[vbno] = c.vbPlasmaStoreMap[vbno].NewWriter()

		return nil
	}

	if vbFlog.streamReqRetry && vbFlog.vb == vbno {

		if vbFlog.statusCode == mcd.ROLLBACK {
			logging.Infof("CRDP[%s:%s:%s:%d] vb: %d vbuuid: %d Rollback requested by DCP, previous startseq: %d rollback startseq: %d",
				c.app.AppName, c.workerName, c.tcpPort, c.Pid(), vbno, vbBlob.VBuuid, start, vbFlog.seqNo)
			start, snapStart, snapEnd = vbFlog.seqNo, vbFlog.seqNo, vbFlog.seqNo
		}

		logging.Infof("CRDP[%s:%s:%s:%d] Retrying DCP stream start vb: %d vbuuid: %d startSeq: %d snapshotStart: %d snapshotEnd: %d",
			c.app.AppName, c.workerName, c.tcpPort, c.Pid(), vbno, vbBlob.VBuuid, start, snapStart, snapEnd)
		dcpFeed.DcpRequestStream(vbno, opaque, flags, vbBlob.VBuuid, start, end, snapStart, snapEnd)
		goto loop
	}

	return nil
}

func (c *Consumer) getCurrentlyOwnedVbs() []int {
	var vbsOwned []int

	for vbNo := 0; vbNo < numVbuckets; vbNo++ {
		if c.vbProcessingStats.getVbStat(uint16(vbNo), "current_vb_owner") == c.HostPortAddr() &&
			c.vbProcessingStats.getVbStat(uint16(vbNo), "assigned_worker") == c.ConsumerName() &&
			c.vbProcessingStats.getVbStat(uint16(vbNo), "node_uuid") == c.NodeUUID() {

			vbsOwned = append(vbsOwned, vbNo)
		}
	}
	return vbsOwned
}
