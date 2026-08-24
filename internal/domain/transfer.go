package domain

import "time"

type CustodyTransfer struct {
	ID                  string    `json:"id"`
	DossierID           string    `json:"dossier_id"`
	Sequence            int       `json:"sequence"`
	StationCode         string    `json:"station_code"`
	ReleasedBy          string    `json:"released_by"`
	ReceivedBy          string    `json:"received_by"`
	TransferredAt       time.Time `json:"transferred_at"`
	ObservedTemperature float64   `json:"observed_temperature"`
	SealState           SealState `json:"seal_state"`
	RequestID           string    `json:"request_id"`
	ContentDigest       string    `json:"content_digest"`
}

type TransferInput struct {
	StationCode         string    `json:"station_code"`
	ReleasedBy          string    `json:"released_by"`
	ReceivedBy          string    `json:"received_by"`
	TransferredAt       time.Time `json:"transferred_at"`
	ObservedTemperature float64   `json:"observed_temperature"`
	SealState           SealState `json:"seal_state"`
}

func NewTransfer(id, dossierID, requestID, digest string, sequence int, input TransferInput) (*CustodyTransfer, error) {
	if NormalizeStation(input.StationCode) == "" {
		return nil, FieldError("station_code", "交接站点不能为空")
	}
	if NormalizeText(input.ReleasedBy) == "" {
		return nil, FieldError("released_by", "交出人不能为空")
	}
	if NormalizeText(input.ReceivedBy) == "" {
		return nil, FieldError("received_by", "接收人不能为空")
	}
	if NormalizeText(input.ReleasedBy) == NormalizeText(input.ReceivedBy) {
		return nil, FieldError("received_by", "交出人与接收人不能相同")
	}
	if input.TransferredAt.IsZero() {
		return nil, FieldError("transferred_at", "交接时间不能为空")
	}
	if input.SealState != SealIntact && input.SealState != SealBroken && input.SealState != SealMissing {
		return nil, FieldError("seal_state", "封签状态无效")
	}
	if sequence <= 0 {
		return nil, FieldError("sequence", "交接序号无效")
	}
	return &CustodyTransfer{ID: id, DossierID: dossierID, Sequence: sequence, StationCode: NormalizeStation(input.StationCode),
		ReleasedBy: NormalizeText(input.ReleasedBy), ReceivedBy: NormalizeText(input.ReceivedBy), TransferredAt: NormalizeTime(input.TransferredAt),
		ObservedTemperature: input.ObservedTemperature, SealState: input.SealState, RequestID: requestID, ContentDigest: digest}, nil
}
