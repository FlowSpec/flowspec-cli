/**
 * @ServiceSpec
 * operationId: "getUser"
 * description: "根据ID获取用户"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "GET"]}
 *   "has_user_id": {"!=": [{"var": "user_id"}, null]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 *   "ok_status": {"==": [{"var": "http_status_code"}, 200]}
 */
export class SuccessService {
    async getUser(userId: string): Promise<User> {
        return this.userRepository.findById(userId);
    }
}
