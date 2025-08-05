/**
 * @ServiceSpec
 * operationId: "createUser"
 * description: "创建新用户账户"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "POST"]}
 *   "has_email": {"!=": [{"var": "request_email"}, null]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 *   "created_status": {"==": [{"var": "http_status_code"}, 201]}
 */
public class SuccessService {
    public User createUser(CreateUserRequest request) {
        return userRepository.save(new User(request));
    }
}
