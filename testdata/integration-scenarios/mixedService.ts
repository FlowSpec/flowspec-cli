/**
 * @ServiceSpec
 * operationId: "mixedGetUser"
 * description: "混合场景 - 应该成功"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "GET"]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 */
/**
 * @ServiceSpec
 * operationId: "mixedDeleteUser"
 * description: "混合场景 - 应该失败 (前置条件)"
 * preconditions:
 *   "required_admin_role": {"==": [{"var": "user_role"}, "admin"]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 */
export class MixedService {
    async mixedGetUser(userId: string): Promise<User> { return new User(); }
    async mixedDeleteUser(userId: string): Promise<void> { }
}
