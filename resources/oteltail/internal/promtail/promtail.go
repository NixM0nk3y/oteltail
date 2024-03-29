package promtail

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/prometheus/common/model"

	"oteltail/internal/config"
	"oteltail/internal/logger"
)

const (
	reservedLabelTenantID = "__tenant_id__"

	userAgent = "lambda-promtail"

	// We use snappy-encoded protobufs over http by default.
	contentType = "application/x-protobuf"

	maxErrMsgLen = 1024
)

type entry struct {
	labels model.LabelSet
	entry  logproto.Entry
}

type batch struct {
	streams map[string]*logproto.Stream
	size    int
	client  Client
}

type batchIf interface {
	add(ctx context.Context, e entry) error
	encode() ([]byte, int, error)
	createPushRequest() (*logproto.PushRequest, int)
	flushBatch(ctx context.Context) error
}

func newBatch(ctx context.Context, pClient Client, entries ...entry) (*batch, error) {
	b := &batch{
		streams: map[string]*logproto.Stream{},
		client:  pClient,
	}

	for _, entry := range entries {
		if err := b.add(ctx, entry); err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (b *batch) add(ctx context.Context, e entry) error {
	labels := labelsMapToString(e.labels, reservedLabelTenantID)
	stream, ok := b.streams[labels]
	if !ok {
		b.streams[labels] = &logproto.Stream{
			Labels:  labels,
			Entries: []logproto.Entry{},
		}
		stream = b.streams[labels]
	}

	stream.Entries = append(stream.Entries, e.entry)
	b.size += len(e.entry.Line)

	if b.size > config.GetConfig(ctx).BatchSize {
		return b.flushBatch(ctx)
	}

	return nil
}

func labelsMapToString(ls model.LabelSet, without ...model.LabelName) string {
	lstrs := make([]string, 0, len(ls))
Outer:
	for l, v := range ls {
		for _, w := range without {
			if l == w {
				continue Outer
			}
		}
		lstrs = append(lstrs, fmt.Sprintf("%s=%q", l, v))
	}

	sort.Strings(lstrs)
	return fmt.Sprintf("{%s}", strings.Join(lstrs, ", "))
}

func (b *batch) encode() ([]byte, int, error) {
	req, entriesCount := b.createPushRequest()
	buf, err := proto.Marshal(req)
	if err != nil {
		return nil, 0, err
	}

	buf = snappy.Encode(nil, buf)
	return buf, entriesCount, nil
}

func (b *batch) createPushRequest() (*logproto.PushRequest, int) {
	req := logproto.PushRequest{
		Streams: make([]logproto.Stream, 0, len(b.streams)),
	}

	entriesCount := 0
	for _, stream := range b.streams {
		req.Streams = append(req.Streams, *stream)
		entriesCount += len(stream.Entries)
	}
	return &req, entriesCount
}

func (b *batch) flushBatch(ctx context.Context) error {
	if b.client != nil {
		err := b.client.sendToOtel(ctx, b)
		if err != nil {
			return err
		}
	}
	b.resetBatch()

	return nil
}

func (b *batch) resetBatch() {
	b.streams = make(map[string]*logproto.Stream)
	b.size = 0
}

func (c *OtelClient) sendToOtel(ctx context.Context, b *batch) error {

	log := logger.GetLogger(ctx)

	buf, _, err := b.encode()
	if err != nil {
		return err
	}

	backoff := backoff.New(ctx, *c.Config.Backoff)
	var status int
	for {
		// send uses `timeout` internally, so `context.Background` is good enough.
		status, err = c.send(ctx, buf)

		// Only retry 429s, 500s and connection-level errors.
		if status > 0 && status != 429 && status/100 != 5 {
			break
		}
		log.Error(fmt.Sprintf("error sending batch, will retry, status: %d", status), "error", err)
		backoff.Wait()

		// Make sure it sends at least once before checking for retry.
		if !backoff.Ongoing() {
			break
		}
	}

	if err != nil {
		log.Error("Failed to send logs!", "error", err)
		return err
	}

	return nil
}

func (c *OtelClient) send(ctx context.Context, buf []byte) (int, error) {

	log := logger.GetLogger(ctx)

	log.InfoContext(ctx, "sending")

	return 200, nil
}
