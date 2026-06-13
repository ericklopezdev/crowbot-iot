package api

import (
	"net/http"

	"github.com/ErickLopezDev/cwlb-server/internal/auth"
	"github.com/ErickLopezDev/cwlb-server/internal/store"
	"github.com/google/uuid"
)

// handleClaimDevice binds an unclaimed device to the account and provisions its
// MQTT credentials (returned once so they can be flashed onto the device).
func (s *Server) handleClaimDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SerialNumber   string `json:"serial_number"`
		ActivationCode string `json:"activation_code"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	serial := req.SerialNumber
	dev, err := s.q.GetDeviceBySerial(r.Context(), &serial)
	if err != nil {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	if dev.AccountID.Valid {
		writeError(w, http.StatusConflict, "device already claimed")
		return
	}
	if dev.ActivationCodeHash == nil || !auth.CheckPassword(*dev.ActivationCodeHash, req.ActivationCode) {
		writeError(w, http.StatusUnauthorized, "invalid activation code")
		return
	}

	mqttUser := dev.ID.String()
	mqttPass, err := randomToken(24)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provision credentials")
		return
	}
	mqttHash, err := auth.HashPassword(mqttPass)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provision credentials")
		return
	}

	aid := accountID(r)
	claimed, err := s.q.ClaimDevice(r.Context(), store.ClaimDeviceParams{
		ID:               dev.ID,
		AccountID:        pgUUID(aid),
		MqttUsername:     &mqttUser,
		MqttPasswordHash: &mqttHash,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "claim device")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"device": toDeviceView(claimed),
		"mqtt": map[string]string{
			"username": mqttUser,
			"password": mqttPass,
			"note":     "shown once — flash these onto the device",
		},
	})
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	devs, err := s.q.ListDevicesByAccount(r.Context(), pgUUID(accountID(r)))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list devices")
		return
	}
	views := make([]deviceView, 0, len(devs))
	for _, d := range devs {
		views = append(views, toDeviceView(d))
	}
	writeJSON(w, http.StatusOK, views)
}

// handleAssignDevice makes a child the active user of a device (one active
// assignment per device, enforced in a transaction).
func (s *Server) handleAssignDevice(w http.ResponseWriter, r *http.Request) {
	deviceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid device id")
		return
	}
	var req struct {
		ChildID string `json:"child_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	childID, err := uuid.Parse(req.ChildID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid child_id")
		return
	}

	aid := accountID(r)
	dev, err := s.q.GetDeviceByID(r.Context(), deviceID)
	if err != nil || !sameUUID(dev.AccountID, aid) {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	child, err := s.q.GetChild(r.Context(), childID)
	if err != nil || child.AccountID != aid {
		writeError(w, http.StatusNotFound, "child not found")
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "assign")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := s.q.WithTx(tx)
	if err := qtx.DeactivateDeviceAssignments(r.Context(), deviceID); err != nil {
		writeError(w, http.StatusInternalServerError, "assign")
		return
	}
	asg, err := qtx.AssignDevice(r.Context(), store.AssignDeviceParams{DeviceID: deviceID, ChildID: childID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "assign")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "assign")
		return
	}
	writeJSON(w, http.StatusOK, asg)
}
