package handler

import (
	"encoding/json"
	"net/http"

	"github.com/r3tr056/go-videoconf/room-service/internal/room"
	"github.com/sirupsen/logrus"
)

type RoomHTTPHandler struct {
	roomManager *room.RoomManager
}

func NewRoomHandler(manager *room.RoomManager) *RoomHTTPHandler {
	return &RoomHTTPHandler{roomManager: manager}
}

func (h *RoomHTTPHandler) CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		Name            string `json:"name"`
		Capacity        int32  `json:"capacity"`
		Password        string `json:"password"`
		CreatedByUserId string `json:"createdByUserId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalud request body", http.StatusBadRequest)
		return
	}

	newRoom, err := h.roomManager.CreateRoom(r.Context(), reqBody.Name, reqBody.Capacity, reqBody.Password, reqBody.CreatedByUserId)
	if err != nil {
		logrus.Errorf("HTTP CreateRoom failed: %v", err)
		http.Error(w, "Failed to create room", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newRoom)
}

func (h *RoomHTTPHandler) GetRoomHandler(w http.ResponseWriter, r *http.Request, roomID string) {
	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	roomDetails, err := h.roomManager.GetRoom(r.Context(), roomID)
	if err != nil {
		logrus.Errorf("HTTP GetRoom failed: %v", err)
		http.Error(w, "Failed to get room", http.StatusInternalServerError)
		return
	}

	if roomDetails == nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roomDetails)
}

func (h *RoomHTTPHandler) UpdateRoomHandler(w http.ResponseWriter, r *http.Request, roomID string) {
	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	var reqBody struct {
		Name     *string `json:"name,omitempty"`
		Capacity *int32  `json:"capacity,omitempty"`
		Password *string `json:"password,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedRoom, err := h.roomManager.UpdateRoom(r.Context(), roomID, reqBody.Name, reqBody.Capacity, reqBody.Password)
	if err != nil {
		logrus.Errorf("HTTP UpdateRoom failed: %v", err)
		http.Error(w, "Failed to update room", http.StatusInternalServerError)
		return
	}

	if updatedRoom == nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedRoom)
}

func (h *RoomHTTPHandler) DeleteRoomHandler(w http.ResponseWriter, r *http.Request, roomID string) {
	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	if err := h.roomManager.DeleteRoom(r.Context(), roomID); err != nil {
		logrus.Errorf("HTTP DeleteRoom failed: %v", err)
		http.Error(w, "Failed to delete room", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

func (h *RoomHTTPHandler) AddParticipantHandler(w http.ResponseWriter, r *http.Request, roomID string) {
	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	var reqBody struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	participant, err := h.roomManager.AddParticipant(r.Context(), roomID, reqBody.UserID, reqBody.Role)
	if err != nil {
		logrus.Errorf("HTTP AddParticipant failed: %v", err)
		http.Error(w, "Failed to add participant", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participant)
}

func (h *RoomHTTPHandler) RemoveParticipantHandler(w http.ResponseWriter, r *http.Request, participantID string) {
	if participantID == "" {
		http.Error(w, "Participant ID is required", http.StatusBadRequest)
		return
	}

	if err := h.roomManager.RemoveParticipant(r.Context(), participantID); err != nil {
		logrus.Errorf("HTTP RemoveParticipant failed: %v", err)
		http.Error(w, "Failed to remove participant", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

func (h *RoomHTTPHandler) ListParticipantsHandler(w http.ResponseWriter, r *http.Request, roomID string) {
	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	participants, err := h.roomManager.ListParticipants(r.Context(), roomID)
	if err != nil {
		logrus.Errorf("HTTP ListParticipants failed: %v", err)
		http.Error(w, "Failed to list participants", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participants)
}
