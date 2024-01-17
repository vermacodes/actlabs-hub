package entity

type AssignmentStatus = string

const (
	AssignmentStatusCreated    AssignmentStatus = "Created"
	AssignmentStatusCompleted  AssignmentStatus = "Completed"
	AssignmentStatusCancelled  AssignmentStatus = "Cancelled"
	AssignmentStatusInProgress AssignmentStatus = "InProgress"
	AssignmentStatusDeleted    AssignmentStatus = "Deleted"
)

type Assignment struct {
	PartitionKey string           `json:"PartitionKey"`
	RowKey       string           `json:"RowKey"`
	AssignmentId string           `json:"assignmentId"`
	UserId       string           `json:"userId"`
	LabId        string           `json:"labId"`
	CreatedBy    string           `json:"createdBy"`
	DeletedBy    string           `json:"deletedBy"`
	CreatedAt    string           `json:"createdAt"`
	StartedAt    string           `json:"startedAt"`
	CompletedAt  string           `json:"completedAt"`
	DeletedAt    string           `json:"deletedAt"`
	Status       AssignmentStatus `json:"status"`
}

type BulkAssignment struct {
	UserIds []string `json:"userIds"`
	LabIds  []string `json:"labIds"`
}

type AssignmentService interface {
	// GetAllLabsRedacted retrieves all labs assigned to a user, with sensitive information redacted.
	// Returns an array of LabType (with redacted information) and any error encountered.
	GetAllLabsRedacted() ([]LabType, error)

	// GetMyAssignedLabs retrieves all labs assigned to a specific user.
	// userId: The ID of the user.
	// Returns an array of LabType and any error encountered.
	GetAssignedLabsRedactedByUserId(userId string) ([]LabType, error)

	// GetAllAssignments retrieves all available assignments.
	// Returns an array of assignments and any error encountered.
	GetAllAssignments() ([]Assignment, error)

	// GetAssignmentsByLabId retrieves assignments associated with a specific lab.
	// labId: The ID of the lab.
	// Returns an array of assignments and any error encountered.
	GetAssignmentsByLabId(labId string) ([]Assignment, error)

	// GetAssignmentsByUserId retrieves assignments associated with a specific user.
	// userId: The ID of the user.
	// Returns an array of assignments and any error encountered.
	GetAssignmentsByUserId(userId string) ([]Assignment, error)

	// CreateAssignments creates new assignments for a set of users and labs.
	// userIds: The IDs of the users.
	// labIds: The IDs of the labs.
	// createdBy: The ID of the user who created the assignments.
	// Returns any error encountered.
	CreateAssignments(userIds []string, labIds []string, createdBy string) error

	// UpdateAssignment updates a set of assignment.
	// userId : The ID of the user.
	// labId : The ID of the lab.
	// status: The new status of the assignment.
	// Returns any error encountered.
	UpdateAssignment(userId string, labId string, status string) error

	// DeleteAssignments deletes a set of assignments.
	// assignmentIds: The IDs of the assignments to delete.
	// Returns any error encountered.
	DeleteAssignments(assignmentIds []string, userPrincipal string) error
}

type AssignmentRepository interface {
	// GetAllAssignments retrieves all available assignments.
	// Returns an array of assignments and any error encountered.
	GetAllAssignments() ([]Assignment, error)

	// GetAssignmentsByLabId retrieves assignments associated with a specific lab.
	// labId: The ID of the lab.
	// Returns an array of assignments and any error encountered.
	GetAssignmentsByLabId(labId string) ([]Assignment, error)

	// GetAssignmentsByUserId retrieves assignments associated with a specific user.
	// userId: The ID of the user.
	// Returns an array of assignments and any error encountered.
	GetAssignmentsByUserId(userId string) ([]Assignment, error)

	// DeleteAssignment deletes a specific assignment.
	// assignmentId: The ID of the assignment to delete.
	// Returns any error encountered.
	DeleteAssignment(assignmentId string) error

	// UpsertAssignment inserts or updates an assignment.
	// assignment: The assignment to insert or update.
	// Returns any error encountered.
	UpsertAssignment(assignment Assignment) error

	// ValidateUser checks if a user is valid.
	// userId: The ID of the user to validate.
	// Returns a boolean indicating if the user is valid and any error encountered.
	ValidateUser(userId string) (bool, error)
}
