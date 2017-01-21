package watcher

import (
	"github.com/golang/mock/gomock"
	"github.com/ryotarai/spotscaler/config"
	"github.com/ryotarai/spotscaler/ec2"
	"github.com/ryotarai/spotscaler/state"
	"testing"
)

func TestUpdateStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := NewMockState(ctrl)

	mock.EXPECT().UpdateStatus(&state.Status{
		CurrentOndemandCapacity: 10,
		CurrentSpotCapacity:     20,
	})

	w := &Watcher{
		State: mock,
		Config: &config.Config{
			CapacityTagKey: "Capacity",
		},
	}

	instances := ec2.Instances{
		{
			Tags: map[string]string{
				"Capacity": "10",
			},
			Lifecycle: "normal",
		},
		{
			Tags: map[string]string{
				"Capacity": "20",
			},
			Lifecycle: "spot",
		},
	}

	err := w.UpdateStatus(instances)
	if err != nil {
		t.Error(err)
	}
}
