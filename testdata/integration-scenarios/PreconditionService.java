/**
 * @ServiceSpec
 * operationId: "strictCreateUser"
 * description: "严格的用户创建，要求特定前置条件"
 * preconditions:
 *   "required_email": {"!=": [{"var": "request_email"}, null]}
 *   "valid_email_format": {"regex": [{"var": "request_email"}, "^[\\w\\.-]+@[\\w\\.-]+\\.[a-zA-Z]{2,}$"]}
 *   "required_method": {"==": [{"var": "http_method"}, "POST"]}
 *   "min_password_length": {">=": [{"strlen": [{"var": "request_password"}]}, 8]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 */
public class PreconditionService {
    public User strictCreateUser(CreateUserRequest request) {
        return userRepository.save(new User(request));
    }
}
