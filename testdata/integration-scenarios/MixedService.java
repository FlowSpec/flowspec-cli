/**
 * @ServiceSpec
 * operationId: "mixedCreateUser"
 * description: "混合场景 - 应该成功"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "POST"]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 */
/**
 * @ServiceSpec
 * operationId: "mixedUpdateUser"
 * description: "混合场景 - 应该失败 (后置条件)"
 * preconditions:
 *   "valid_method": {"==": [{"var": "http_method"}, "PUT"]}
 * postconditions:
 *   "success_status": {"==": [{"var": "span.status.code"}, "OK"]}
 */
/**
 * @ServiceSpec
 * operationId: "mixedNonExistentOperation"
 * description: "混合场景 - 应该跳过 (无对应轨迹)"
 * preconditions:
 *   "always_true": {"==": [true, true]}
 * postconditions:
 *   "always_true": {"==": [true, true]}
 */
public class MixedService {
    public User mixedCreateUser(CreateUserRequest request) { return new User(); }
    public User mixedUpdateUser(String id, UpdateUserRequest request) { return new User(); }
}
