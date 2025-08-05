/**
 * @ServiceSpec
 * operationId: "strictGetUser"
 * description: "严格的用户获取，要求特定前置条件"
 * preconditions:
 *   "required_user_id": {"!=": [{"var": "user_id"}, null]}
 *   "valid_user_id_format": {"regex": [{"var": "user_id"}, "^[a-zA-Z0-9]{8,}$"]}
 *   "required_method": {"==": [{"var": "http_method"}, "GET"]}
 *   "has_auth_token": {"!=": [{"var": "auth_token"}, null]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 */
export class PreconditionService {
    async strictGetUser(userId: string): Promise<User> {
        return this.userRepository.findById(userId);
    }
}
