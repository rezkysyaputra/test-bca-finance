package main

import (
	"errors"
	"time"
)

type UserRole string

const (
	RoleSales    UserRole = "SALES"
	RoleApprover UserRole = "APPROVER"
	RoleAdmin    UserRole = "ADMIN"
)

type AppStatus string

const (
	StatusDraft     AppStatus = "DRAFT"
	StatusSubmitted AppStatus = "SUBMITTED"
	StatusApproved  AppStatus = "APPROVED"
	StatusRejected  AppStatus = "REJECTED"
)

var (
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrUnauthorized      = errors.New("unauthorized action for this role")
	ErrNotFound          = errors.New("resource not found")
)

func ValidateStatusTransition(current, target AppStatus) error {
	switch current {
	case StatusDraft:
		if target != StatusSubmitted {
			return ErrInvalidTransition
		}
	case StatusSubmitted:
		if target != StatusApproved && target != StatusRejected {
			return ErrInvalidTransition
		}
	default:
		return ErrInvalidTransition
	}
	return nil
}

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Customer struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	NIK       string    `json:"nik"`
	Phone     string    `json:"phone"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
}

type Vehicle struct {
	ID        int64     `json:"id"`
	Dealer    string    `json:"dealer"`
	Brand     string    `json:"brand"`
	Model     string    `json:"model"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

type Application struct {
	ID          int64     `json:"id"`
	CustomerID  int64     `json:"customer_id"`
	VehicleID   int64     `json:"vehicle_id"`
	CreatedBy   int64     `json:"created_by"`
	DownPayment float64   `json:"down_payment"`
	Tenor       int       `json:"tenor"`
	Installment float64   `json:"installment"`
	Status      AppStatus `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Customer  *Customer             `json:"customer,omitempty"`
	Vehicle   *Vehicle              `json:"vehicle,omitempty"`
	Documents []ApplicationDocument `json:"documents,omitempty"`
	Approvals []ApplicationApproval `json:"approvals,omitempty"`
}

type ApplicationDocument struct {
	ID            int64     `json:"id"`
	ApplicationID int64     `json:"application_id"`
	DocumentType  string    `json:"document_type"`
	FilePath      string    `json:"file_path"`
	CreatedAt     time.Time `json:"created_at"`
}

type ApplicationApproval struct {
	ID            int64     `json:"id"`
	ApplicationID int64     `json:"application_id"`
	ApproverID    int64     `json:"approver_id"`
	ApproverName  string    `json:"approver_name,omitempty"`
	Status        AppStatus `json:"status"`
	Notes         string    `json:"notes"`
	ApprovedAt    time.Time `json:"approved_at"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

type UserInfo struct {
	ID       int64    `json:"id"`
	Username string   `json:"username"`
	Role     UserRole `json:"role"`
}

type CreateApplicationRequest struct {
	CustomerName    string  `json:"customer_name" binding:"required"`
	CustomerNIK     string  `json:"customer_nik" binding:"required"`
	CustomerPhone   string  `json:"customer_phone"`
	CustomerAddress string  `json:"customer_address"`
	VehicleID       int64   `json:"vehicle_id" binding:"required"`
	DownPayment     float64 `json:"down_payment" binding:"required"`
	Tenor           int     `json:"tenor" binding:"required"`
	Installment     float64 `json:"installment"`
}

type ApprovalRequest struct {
	Notes string `json:"notes"`
}
