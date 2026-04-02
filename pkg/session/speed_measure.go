package session

import "time"

// ── SendingFile 속도 메서드 ───────────────────────────────────────────────

func (sf *SendingFile) RecordSpeedSample() {
	now := time.Now()
	currentBytes := sf.Stream.BytesSent()

	elapsed := now.Sub(sf.LastSnapshot.Timestamp).Seconds()
	if elapsed <= 0 {
		return
	}

	speed := int64(float64(currentBytes-sf.LastSnapshot.Bytes) / elapsed)

	sf.SpeedMu.Lock()
	sf.SpeedHistory = append(sf.SpeedHistory, speed)
	if len(sf.SpeedHistory) > speedHistorySize {
		sf.SpeedHistory = sf.SpeedHistory[1:]
	}
	sf.SpeedMu.Unlock()

	sf.LastSnapshot = ByteSnapshot{
		Bytes:     currentBytes,
		Timestamp: now,
	}
}

func (sf *SendingFile) CurrentSpeed() int64 {
	sf.SpeedMu.Lock()
	defer sf.SpeedMu.Unlock()
	if len(sf.SpeedHistory) == 0 {
		return 0
	}
	return sf.SpeedHistory[len(sf.SpeedHistory)-1]
}

func (sf *SendingFile) AverageSpeed() int64 {
	sf.SpeedMu.Lock()
	defer sf.SpeedMu.Unlock()
	if len(sf.SpeedHistory) == 0 {
		return 0
	}
	var sum int64
	for _, s := range sf.SpeedHistory {
		sum += s
	}
	return sum / int64(len(sf.SpeedHistory))
}

// ── ReceivingFile 속도 메서드 (동일 구조) ────────────────────────────────

func (rf *ReceivingFile) RecordSpeedSample() {
	now := time.Now()
	currentBytes := rf.Stream.BytesReceived()

	elapsed := now.Sub(rf.LastSnapshot.Timestamp).Seconds()
	if elapsed <= 0 {
		return
	}

	speed := int64(float64(currentBytes-rf.LastSnapshot.Bytes) / elapsed)

	rf.SpeedMu.Lock()
	rf.SpeedHistory = append(rf.SpeedHistory, speed)
	if len(rf.SpeedHistory) > speedHistorySize {
		rf.SpeedHistory = rf.SpeedHistory[1:]
	}
	rf.SpeedMu.Unlock()

	rf.LastSnapshot = ByteSnapshot{
		Bytes:     currentBytes,
		Timestamp: now,
	}
}

func (rf *ReceivingFile) CurrentSpeed() int64 {
	rf.SpeedMu.Lock()
	defer rf.SpeedMu.Unlock()
	if len(rf.SpeedHistory) == 0 {
		return 0
	}
	return rf.SpeedHistory[len(rf.SpeedHistory)-1]
}

func (rf *ReceivingFile) AverageSpeed() int64 {
	rf.SpeedMu.Lock()
	defer rf.SpeedMu.Unlock()
	if len(rf.SpeedHistory) == 0 {
		return 0
	}
	var sum int64
	for _, s := range rf.SpeedHistory {
		sum += s
	}
	return sum / int64(len(rf.SpeedHistory))
}
