// SPDX-License-Identifier: AGPL-3.0-only
package desktop

import (
	"context"
	"testing"
)

type expiringQR struct {
	mockSession
	issued int
}

func (q *expiringQR) CreateQR(context.Context) ([]byte, error) {
	q.issued++
	return []byte{byte(q.issued)}, nil
}
func (q *expiringQR) PollLogin(ctx context.Context) (string, error) {
	if q.issued == 1 {
		return "expired", nil
	}
	<-ctx.Done()
	return "", ctx.Err()
}
func TestExpiredQRRefreshesSameTab(t *testing.T) {
	q := &expiringQR{}
	c := newTestController(t, q)
	id := c.Snapshot().Active
	if err := c.Start(id); err != nil {
		t.Fatal(err)
	}
	a := awaitView(t, c, id, func(v Account) bool { return len(v.QR) == 1 && v.QR[0] == 2 })
	if a.Status != "scan" || a.SignedIn || a.DraftEpoch != 0 {
		t.Fatal("renewal reset or authorized the tab")
	}
}
