/**
 * @ServiceSpec
 * operationId: "createUser"
 * description: "创建新用户账户"
 * preconditions: {
 *   "request.body.email": {"!=": null}
 * }
 * postconditions: {
 *   "response.status": {"==": 201}
 * }
 */
public User createUser(CreateUserRequest request) {
    // 实现代码
    return new User();
}
