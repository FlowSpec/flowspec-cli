/**
 * @ServiceSpec
 * operationId: "expectSuccessGetUser"
 * description: "期望成功的用户获取"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "GET"]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 *   "ok_status": {"==": [{"var": "http_status_code"}, 200]}
 *   "has_user_data": {"!=": [{"var": "response_user"}, null]}
 */
export class PostconditionService {
    async expectSuccessGetUser(userId: string): Promise<User> {
        return this.userRepository.findById(userId);
    }
}
