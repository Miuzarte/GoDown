package main

import (
	"errors"
	"fmt"
)

var (
	// General errors
	ErrInternal  = errors.New("Internal error occured")
	ErrArgs      = errors.New("Invalid arguments")
	ErrAgain     = errors.New("Try again")
	ErrRateLimit = errors.New("Rate limit reached")
	ErrBadResp   = errors.New("Bad response from server")

	// Upload errors
	ErrFailed  = errors.New("The upload failed. Please restart it from scratch")
	ErrTooMany = errors.New("Too many concurrent IP addresses are accessing this upload target URL")
	ErrRange   = errors.New("The upload file packet is out of range or not starting and ending on a chunk boundary")
	ErrExpired = errors.New("The upload target URL you are trying to access has expired. Please request a fresh one")

	// Filesystem/Account errors
	ErrNoEnt              = errors.New("Object (typically, node or user) not found")
	ErrCircular           = errors.New("Circular linkage attempted")
	ErrAccess             = errors.New("Access violation")
	ErrExist              = errors.New("Trying to create an object that already exists")
	ErrIncomplete         = errors.New("Trying to access an incomplete resource")
	ErrKey                = errors.New("A decryption operation failed")
	ErrSid                = errors.New("Invalid or expired user session, please relogin")
	ErrBlocked            = errors.New("User blocked")
	ErrOverQuota          = errors.New("Request over quota")
	ErrTempUnavail        = errors.New("Resource temporarily not available, please try again later")
	ErrMacMismatch        = errors.New("MAC verification failed")
	ErrBadAttr            = errors.New("Bad node attribute")
	ErrTooManyConnections = errors.New("Too many connections on this resource.")
	ErrWrite              = errors.New("File could not be written to (or failed post-write integrity check).")
	ErrRead               = errors.New("File could not be read from (or changed unexpectedly during reading).")
	ErrAppKey             = errors.New("Invalid or missing application key.")
	ErrSsl                = errors.New("SSL verification failed")
	ErrGoingOverQuota     = errors.New("Not enough quota")
	ErrMfRequired         = errors.New("Multi-factor authentication required")

	// Config errors
	// EWORKER_LIMIT_EXCEEDED = errors.New("Maximum worker limit exceeded")
)

type ErrorMsg int

func parseError(errno ErrorMsg) error {
	switch {
	case errno == 0:
		return nil
	case errno == -1:
		return ErrInternal
	case errno == -2:
		return ErrArgs
	case errno == -3:
		return ErrAgain
	case errno == -4:
		return ErrRateLimit
	case errno == -5:
		return ErrFailed
	case errno == -6:
		return ErrTooMany
	case errno == -7:
		return ErrRange
	case errno == -8:
		return ErrExpired
	case errno == -9:
		return ErrNoEnt
	case errno == -10:
		return ErrCircular
	case errno == -11:
		return ErrAccess
	case errno == -12:
		return ErrExist
	case errno == -13:
		return ErrIncomplete
	case errno == -14:
		return ErrKey
	case errno == -15:
		return ErrSid
	case errno == -16:
		return ErrBlocked
	case errno == -17:
		return ErrOverQuota
	case errno == -18:
		return ErrTempUnavail
	case errno == -19:
		return ErrTooManyConnections
	case errno == -20:
		return ErrWrite
	case errno == -21:
		return ErrRead
	case errno == -22:
		return ErrAppKey
	case errno == -23:
		return ErrSsl
	case errno == -24:
		return ErrGoingOverQuota
	case errno == -26:
		return ErrMfRequired
	}

	return fmt.Errorf("Unknown mega error %d", errno)
}
