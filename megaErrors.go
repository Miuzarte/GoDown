package main

import (
	"errors"
	"fmt"
)

var (
	// General errors
	ErrInternal  = errors.New("internal error occured")
	ErrArgs      = errors.New("invalid arguments")
	ErrAgain     = errors.New("try again")
	ErrRateLimit = errors.New("rate limit reached")
	ErrBadResp   = errors.New("bad response from server")

	// Upload errors
	ErrFailed  = errors.New("the upload failed. Please restart it from scratch")
	ErrTooMany = errors.New("too many concurrent IP addresses are accessing this upload target URL")
	ErrRange   = errors.New("the upload file packet is out of range or not starting and ending on a chunk boundary")
	ErrExpired = errors.New("the upload target URL you are trying to access has expired. Please request a fresh one")

	// Filesystem/Account errors
	ErrNoEnt              = errors.New("object (typically, node or user) not found")
	ErrCircular           = errors.New("circular linkage attempted")
	ErrAccess             = errors.New("access violation")
	ErrExist              = errors.New("trying to create an object that already exists")
	ErrIncomplete         = errors.New("trying to access an incomplete resource")
	ErrKey                = errors.New("a decryption operation failed")
	ErrSid                = errors.New("invalid or expired user session, please relogin")
	ErrBlocked            = errors.New("user blocked")
	ErrOverQuota          = errors.New("request over quota")
	ErrTempUnavail        = errors.New("resource temporarily not available, please try again later")
	ErrMacMismatch        = errors.New("MAC verification failed")
	ErrBadAttr            = errors.New("bad node attribute")
	ErrTooManyConnections = errors.New("too many connections on this resource")
	ErrWrite              = errors.New("file could not be written to (or failed post-write integrity check)")
	ErrRead               = errors.New("file could not be read from (or changed unexpectedly during reading)")
	ErrAppKey             = errors.New("invalid or missing application key")
	ErrSsl                = errors.New("SSL verification failed")
	ErrGoingOverQuota     = errors.New("not enough quota")
	ErrMfRequired         = errors.New("multi-factor authentication required")
)

type ErrorMsg int

func (e ErrorMsg) Parse() error {
	switch e {
	case 0:
		return nil
	case -1:
		return ErrInternal
	case -2:
		return ErrArgs
	case -3:
		return ErrAgain
	case -4:
		return ErrRateLimit
	case -5:
		return ErrFailed
	case -6:
		return ErrTooMany
	case -7:
		return ErrRange
	case -8:
		return ErrExpired
	case -9:
		return ErrNoEnt
	case -10:
		return ErrCircular
	case -11:
		return ErrAccess
	case -12:
		return ErrExist
	case -13:
		return ErrIncomplete
	case -14:
		return ErrKey
	case -15:
		return ErrSid
	case -16:
		return ErrBlocked
	case -17:
		return ErrOverQuota
	case -18:
		return ErrTempUnavail
	case -19:
		return ErrTooManyConnections
	case -20:
		return ErrWrite
	case -21:
		return ErrRead
	case -22:
		return ErrAppKey
	case -23:
		return ErrSsl
	case -24:
		return ErrGoingOverQuota
	case -26:
		return ErrMfRequired
	}
	return fmt.Errorf("unknown mega error: %d", e)
}
