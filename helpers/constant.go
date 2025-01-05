package helpers

type STATUS string

const (
	WAITING     STATUS = "WAITING"
	IN_PROGRESS STATUS = "IN_PROGRESS"
	CALLED      STATUS = "CALLED"
)

const (
	ADMIN   = "Admin"
	STUDENT = "Student"
)
