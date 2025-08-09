package service

import (
	"context"
	"github.com/example/models"
)

type UserService struct {
	repo UserRepository
}

// @ServiceSpec
// operationId: "createUser"
// description: "Create a new user account"
// preconditions:
//
//	"email_validation": {
//	  "and": [
//	    {"!=": [{"var": "request.body.email"}, null]},
//	    {"regex": [{"var": "request.body.email"}, "^[\\w\\.-]+@[\\w\\.-]+\\.[a-zA-Z]{2,}$"]}
//	  ]
//	}
//	"password_strength": {
//	  ">=": [{"strlen": [{"var": "request.body.password"}]}, 8]
//	}
//
// postconditions:
//
//	"successful_creation": {
//	  "and": [
//	    {"==": [{"var": "response.status"}, 201]},
//	    {"!=": [{"var": "response.body.userId"}, null]}
//	  ]
//	}
func (s *UserService) CreateUser(ctx context.Context, request *CreateUserRequest) (*User, error) {
	return s.repo.Save(ctx, models.NewUser(request))
}

/*
 * @ServiceSpec
 * operationId: "getUser"
 * description: "Retrieve user by ID"
 * preconditions:
 *   "valid_user_id": {
 *     "!=": [{"var": "request.path.userId"}, null]
 *   }
 * postconditions:
 *   "successful_retrieval": {
 *     "==": [{"var": "response.status"}, 200]
 *   }
 */
func (s *UserService) GetUser(ctx context.Context, userId string) (*User, error) {
	return s.repo.FindById(ctx, userId)
}
