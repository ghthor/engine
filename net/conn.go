package net

import "github.com/ghthor/filu"

// A EncodedType is used to mark the the following value's
// type to enable decoding into a concrete value go value.
type EncodedType int

//go:generate stringer -type=EncodedType
const (
	ET_ERROR EncodedType = iota
	ET_PROTOCOL_ERROR
	ET_DISCONNECT

	ET_USER_LOGIN_REQUEST

	ET_USER_LOGIN_FAILED
	ET_USER_LOGIN_SUCCESS
	ET_USER_CREATE_SUCCESS

	ET_ACTORS
	ET_SELECT_ACTOR
	ET_SELECT_ACTOR_SUCCESS
	ET_CREATE_ACTOR_SUCCESS

	// Used to entend the EncodedType enumeration in other packages.
	// WARNING: Only reccomended to extend in one place, else
	// the values taken by the enumeration cases could overlap.
	ET_EXTEND
)

type EncodableType interface {
	Type() EncodedType
}

type ProtocolError string

type UserLoginRequest struct{ Name, Password string }
type UserLoginFailure struct{ Name string }
type UserLoginSuccess struct{ Name string }
type UserCreateSuccess UserLoginSuccess

type ActorsList []string
type SelectActorRequest struct{ Name string }
type SelectActorSuccess struct{ Actor filu.Actor }
type CreateActorSuccess struct{ Actor filu.Actor }

const DisconnectResponse = "disconnected"

func (ProtocolError) Type() EncodedType { return ET_PROTOCOL_ERROR }
func (e ProtocolError) Error() string   { return string(e) }
func (e ProtocolError) String() string  { return string(e) }

func (UserLoginRequest) Type() EncodedType  { return ET_USER_LOGIN_REQUEST }
func (UserLoginFailure) Type() EncodedType  { return ET_USER_LOGIN_FAILED }
func (UserLoginSuccess) Type() EncodedType  { return ET_USER_LOGIN_SUCCESS }
func (UserCreateSuccess) Type() EncodedType { return ET_USER_CREATE_SUCCESS }

func (ActorsList) Type() EncodedType         { return ET_ACTORS }
func (SelectActorRequest) Type() EncodedType { return ET_SELECT_ACTOR }
func (SelectActorSuccess) Type() EncodedType { return ET_SELECT_ACTOR_SUCCESS }
func (CreateActorSuccess) Type() EncodedType { return ET_CREATE_ACTOR_SUCCESS }

type Encoder interface {
	Encode(EncodableType) error
}

type Decoder interface {
	NextType() (EncodedType, error)
	Decode(EncodableType) error
}

type Conn interface {
	Encoder
	Decoder
}