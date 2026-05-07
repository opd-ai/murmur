package input

import (
	"testing"
	"time"
)

func TestNormalizeWheelToZoomActions(t *testing.T) {
	t.Parallel()

	mapper := NewMapper()
	now := time.Now()

	zoomIn := mapper.Normalize(RawEvent{Source: SourceMouse, Type: EventWheel, X: 10, Y: 20, Delta: 2, Timestamp: now})
	if len(zoomIn) != 1 || zoomIn[0].Action != ActionZoomIn {
		t.Fatalf("expected one zoom-in action, got %+v", zoomIn)
	}
	if zoomIn[0].Delta != 2 {
		t.Fatalf("expected zoom-in delta 2, got %v", zoomIn[0].Delta)
	}

	zoomOut := mapper.Normalize(RawEvent{Source: SourceMouse, Type: EventWheel, X: 10, Y: 20, Delta: -3, Timestamp: now})
	if len(zoomOut) != 1 || zoomOut[0].Action != ActionZoomOut {
		t.Fatalf("expected one zoom-out action, got %+v", zoomOut)
	}
	if zoomOut[0].Delta != 3 {
		t.Fatalf("expected zoom-out delta 3, got %v", zoomOut[0].Delta)
	}
}

func TestNormalizeMoveToPanAction(t *testing.T) {
	t.Parallel()

	mapper := NewMapper()
	actions := mapper.Normalize(RawEvent{Source: SourceTouch, Type: EventMove, X: 4, Y: -6, Timestamp: time.Now()})
	if len(actions) != 1 {
		t.Fatalf("expected one action, got %d", len(actions))
	}
	if actions[0].Action != ActionPan {
		t.Fatalf("expected ActionPan, got %q", actions[0].Action)
	}
	if actions[0].X != 4 || actions[0].Y != -6 {
		t.Fatalf("unexpected pan payload: %+v", actions[0])
	}
}
