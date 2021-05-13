/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"reflect"
	"testing"
)

func TestBBA_Output(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		input  bool
		want   interface{}
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bba := &BBA{
				Config:        tt.fields.Config,
				Mutex:         tt.fields.Mutex,
				epoch:         tt.fields.epoch,
				binValues:     tt.fields.binValues,
				sentBvals:     tt.fields.sentBvals,
				receivedBvals: tt.fields.receivedBvals,
				receivedAux:   tt.fields.receivedAux,
				done:          tt.fields.done,
				output:        tt.fields.output,
				estimated:     tt.fields.estimated,
				decision:      tt.fields.decision,
				messages:      tt.fields.messages,
			}
			if got := bba.Output(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BBA.Output() = %v, want %v", got, tt.want)
			}
		})
	}
}
