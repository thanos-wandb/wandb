package interfaces

import (
	"encoding/json"
	"sync"

	spb "github.com/wandb/wandb/core/pkg/service_go_proto"
	"github.com/wandb/wandb/experimental/go-sdk/internal/connection"
	"github.com/wandb/wandb/experimental/go-sdk/internal/mailbox"
	"github.com/wandb/wandb/experimental/go-sdk/pkg/runconfig"
	"github.com/wandb/wandb/experimental/go-sdk/pkg/settings"
)

type SockInterface struct {
	Conn     *connection.Connection
	StreamId string
	wg       sync.WaitGroup
}

func (s *SockInterface) Start() {
	s.wg.Add(1)
	go func() {
		s.Conn.Recv()
		s.wg.Done()
	}()
}

func (s *SockInterface) Close() {
	s.Conn.Close()
	s.wg.Wait()
}

func (s *SockInterface) InformInit(settings *settings.Settings) error {
	serverRecord := spb.ServerRequest{
		ServerRequestType: &spb.ServerRequest_InformInit{
			InformInit: &spb.ServerInformInitRequest{
				Settings: settings.ToProto(),
				XInfo: &spb.XRecordInfo{
					StreamId: s.StreamId,
				},
			},
		},
	}
	return s.Conn.Send(&serverRecord)
}

func (s *SockInterface) DeliverRunRecord(
	settings *settings.Settings,
	config *runconfig.Config,
	telemetry *spb.TelemetryRecord,
) (*mailbox.MailboxHandle, error) {

	cfg := &spb.ConfigRecord{}
	for key, value := range *config {
		data, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		cfg.Update = append(cfg.Update, &spb.ConfigItem{
			Key:       key,
			ValueJson: string(data),
		})
	}
	record := spb.Record{
		RecordType: &spb.Record_Run{
			Run: &spb.RunRecord{
				RunId:       settings.RunID,
				DisplayName: settings.RunName,
				Project:     settings.RunProject,
				Config:      cfg,
				Telemetry:   telemetry,
				XInfo: &spb.XRecordInfo{
					StreamId: s.StreamId,
				},
			},
		},
		XInfo: &spb.XRecordInfo{
			StreamId: s.StreamId,
		},
	}
	serverRecord := spb.ServerRequest{
		ServerRequestType: &spb.ServerRequest_RecordCommunicate{
			RecordCommunicate: &record,
		},
	}

	handle := s.Conn.Mailbox.Deliver(&record)
	if err := s.Conn.Send(&serverRecord); err != nil {
		return nil, err
	}
	return handle, nil
}

func (s *SockInterface) InformStart(settings *settings.Settings) error {
	serverRecord := spb.ServerRequest{
		ServerRequestType: &spb.ServerRequest_InformStart{
			InformStart: &spb.ServerInformStartRequest{
				Settings: settings.ToProto(),
				XInfo: &spb.XRecordInfo{
					StreamId: s.StreamId,
				},
			},
		},
	}
	return s.Conn.Send(&serverRecord)
}

func (s *SockInterface) DeliverRunStartRequest(settings *settings.Settings) (*mailbox.MailboxHandle, error) {
	record := spb.Record{
		RecordType: &spb.Record_Request{
			Request: &spb.Request{
				RequestType: &spb.Request_RunStart{
					RunStart: &spb.RunStartRequest{
						Run: &spb.RunRecord{
							RunId: settings.RunID,
						},
					},
				},
			},
		},
		Control: &spb.Control{
			Local: true,
		},
		XInfo: &spb.XRecordInfo{
			StreamId: s.StreamId,
		},
	}

	serverRecord := spb.ServerRequest{
		ServerRequestType: &spb.ServerRequest_RecordCommunicate{
			RecordCommunicate: &record,
		},
	}

	handle := s.Conn.Mailbox.Deliver(&record)
	if err := s.Conn.Send(&serverRecord); err != nil {
		return nil, err
	}
	return handle, nil
}

func (s *SockInterface) PublishPartialHistory(data map[string]interface{}) error {
	history := spb.PartialHistoryRequest{}
	for key, value := range data {
		data, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		history.Item = append(history.Item, &spb.HistoryItem{
			Key:       key,
			ValueJson: string(data),
		})
	}
	record := spb.Record{
		RecordType: &spb.Record_Request{
			Request: &spb.Request{
				RequestType: &spb.Request_PartialHistory{
					PartialHistory: &history,
				},
			},
		},
		Control: &spb.Control{
			Local: true,
		},
		XInfo: &spb.XRecordInfo{
			StreamId: s.StreamId,
		},
	}

	return s.Conn.Send(&record)
}

func (s *SockInterface) DeliverExitRecord() (*mailbox.MailboxHandle, error) {
	record := spb.Record{
		RecordType: &spb.Record_Exit{
			Exit: &spb.RunExitRecord{
				ExitCode: 0,
				XInfo: &spb.XRecordInfo{
					StreamId: s.StreamId,
				},
			},
		},
		XInfo: &spb.XRecordInfo{
			StreamId: s.StreamId,
		},
	}
	serverRecord := spb.ServerRequest{
		ServerRequestType: &spb.ServerRequest_RecordCommunicate{
			RecordCommunicate: &record,
		},
	}
	handle := s.Conn.Mailbox.Deliver(&record)
	if err := s.Conn.Send(&serverRecord); err != nil {
		return nil, err
	}
	return handle, nil
}

func (s *SockInterface) DeliverShutdownRequest() (*mailbox.MailboxHandle, error) {
	record := spb.Record{
		RecordType: &spb.Record_Request{
			Request: &spb.Request{
				RequestType: &spb.Request_Shutdown{
					Shutdown: &spb.ShutdownRequest{},
				},
			}},
		Control: &spb.Control{
			AlwaysSend: true,
			ReqResp:    true,
		},
	}
	serverRecord := spb.ServerRequest{
		ServerRequestType: &spb.ServerRequest_RecordCommunicate{
			RecordCommunicate: &record,
		},
	}
	handle := s.Conn.Mailbox.Deliver(&record)
	if err := s.Conn.Send(&serverRecord); err != nil {
		return nil, err
	}
	return handle, nil
}

func (s *SockInterface) InformFinish() error {
	serverRecord := spb.ServerRequest{
		ServerRequestType: &spb.ServerRequest_InformFinish{
			InformFinish: &spb.ServerInformFinishRequest{
				XInfo: &spb.XRecordInfo{
					StreamId: s.StreamId,
				},
			},
		},
	}
	return s.Conn.Send(&serverRecord)
}
