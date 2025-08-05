// @ServiceSpec
// operationId: "updateUser"
// description: "更新用户信息"
// preconditions:
//   "valid_method": {"==": [{"var": "http_method"}, "PUT"]}
//   "has_user_id": {"!=": [{"var": "user_id"}, null]}
// postconditions:
//   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
//   "ok_status": {"==": [{"var": "http_status_code"}, 200]}
func UpdateUser(userId string, user User) error {
    return userRepository.Update(userId, user)
}
