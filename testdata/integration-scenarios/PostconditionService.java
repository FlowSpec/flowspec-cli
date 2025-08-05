/**
 * @ServiceSpec
 * operationId: "expectSuccessCreateUser"
 * description: "期望成功的用户创建"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "POST"]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 *   "created_status": {"==": [{"var": "http_status_code"}, 201]}
 *   "has_user_id": {"!=": [{"var": "response_user_id"}, null]}
 */
public class PostconditionService {
    public User expectSuccessCreateUser(CreateUserRequest request) {
        return userRepository.save(new User(request));
    }
}
