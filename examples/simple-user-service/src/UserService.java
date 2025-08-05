package com.example.userservice;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.atomic.AtomicLong;

/**
 * 简单的用户管理服务
 * 
 * 这个示例展示了如何在 Java 代码中使用 ServiceSpec 注解
 * 来定义服务契约和验证规则。
 */
public class UserService {
    
    private final Map<Long, User> users = new HashMap<>();
    private final AtomicLong idGenerator = new AtomicLong(1);
    
    /**
     * @ServiceSpec
     * operationId: "createUser"
     * description: "创建新用户账户"
     * preconditions: {
     *   "email_required": {"!=": [{"var": "span.attributes.request.body.email"}, null]},
     *   "email_format": {"match": [{"var": "span.attributes.request.body.email"}, "^[\\w\\.-]+@[\\w\\.-]+\\.[a-zA-Z]{2,}$"]},
     *   "password_length": {">=": [{"var": "span.attributes.request.body.password.length"}, 8]}
     * }
     * postconditions: {
     *   "success_status": {"==": [{"var": "span.attributes.http.status_code"}, 201]},
     *   "user_id_generated": {"!=": [{"var": "span.attributes.response.body.userId"}, null]},
     *   "email_returned": {"==": [{"var": "span.attributes.response.body.email"}, {"var": "span.attributes.request.body.email"}]}
     * }
     */
    public User createUser(CreateUserRequest request) {
        // 验证输入参数
        if (request.getEmail() == null || request.getEmail().isEmpty()) {
            throw new IllegalArgumentException("Email is required");
        }
        
        if (request.getPassword() == null || request.getPassword().length() < 8) {
            throw new IllegalArgumentException("Password must be at least 8 characters");
        }
        
        // 创建新用户
        Long userId = idGenerator.getAndIncrement();
        User user = new User(userId, request.getEmail(), request.getName());
        users.put(userId, user);
        
        return user;
    }
    
    /**
     * @ServiceSpec
     * operationId: "getUser"
     * description: "根据用户ID获取用户信息"
     * preconditions: {
     *   "user_id_required": {"!=": [{"var": "span.attributes.request.params.userId"}, null]},
     *   "user_id_format": {"match": [{"var": "span.attributes.request.params.userId"}, "^[0-9]+$"]}
     * }
     * postconditions: {
     *   "success_or_not_found": {"in": [{"var": "span.attributes.http.status_code"}, [200, 404]]},
     *   "user_data_if_found": {
     *     "if": [
     *       {"==": [{"var": "span.attributes.http.status_code"}, 200]},
     *       {"and": [
     *         {"!=": [{"var": "span.attributes.response.body.userId"}, null]},
     *         {"!=": [{"var": "span.attributes.response.body.email"}, null]}
     *       ]},
     *       true
     *     ]
     *   }
     * }
     */
    public User getUser(Long userId) {
        if (userId == null) {
            throw new IllegalArgumentException("User ID is required");
        }
        
        return users.get(userId);
    }
    
    /**
     * @ServiceSpec
     * operationId: "updateUser"
     * description: "更新用户信息"
     * preconditions: {
     *   "user_id_required": {"!=": [{"var": "span.attributes.request.params.userId"}, null]},
     *   "update_data_provided": {
     *     "or": [
     *       {"!=": [{"var": "span.attributes.request.body.name"}, null]},
     *       {"!=": [{"var": "span.attributes.request.body.email"}, null]}
     *     ]
     *   },
     *   "email_format_if_provided": {
     *     "if": [
     *       {"!=": [{"var": "span.attributes.request.body.email"}, null]},
     *       {"match": [{"var": "span.attributes.request.body.email"}, "^[\\w\\.-]+@[\\w\\.-]+\\.[a-zA-Z]{2,}$"]},
     *       true
     *     ]
     *   }
     * }
     * postconditions: {
     *   "success_or_not_found": {"in": [{"var": "span.attributes.http.status_code"}, [200, 404]]},
     *   "updated_data_if_success": {
     *     "if": [
     *       {"==": [{"var": "span.attributes.http.status_code"}, 200]},
     *       {"and": [
     *         {"!=": [{"var": "span.attributes.response.body.userId"}, null]},
     *         {"==": [{"var": "span.attributes.response.body.userId"}, {"var": "span.attributes.request.params.userId"}]}
     *       ]},
     *       true
     *     ]
     *   }
     * }
     */
    public User updateUser(Long userId, UpdateUserRequest request) {
        if (userId == null) {
            throw new IllegalArgumentException("User ID is required");
        }
        
        User existingUser = users.get(userId);
        if (existingUser == null) {
            return null; // 用户不存在
        }
        
        // 更新用户信息
        if (request.getName() != null) {
            existingUser.setName(request.getName());
        }
        
        if (request.getEmail() != null) {
            existingUser.setEmail(request.getEmail());
        }
        
        return existingUser;
    }
    
    /**
     * @ServiceSpec
     * operationId: "deleteUser"
     * description: "删除用户账户"
     * preconditions: {
     *   "user_id_required": {"!=": [{"var": "span.attributes.request.params.userId"}, null]},
     *   "user_id_format": {"match": [{"var": "span.attributes.request.params.userId"}, "^[0-9]+$"]}
     * }
     * postconditions: {
     *   "success_or_not_found": {"in": [{"var": "span.attributes.http.status_code"}, [204, 404]]},
     *   "no_content_if_deleted": {
     *     "if": [
     *       {"==": [{"var": "span.attributes.http.status_code"}, 204]},
     *       {"==": [{"var": "span.attributes.response.body"}, null]},
     *       true
     *     ]
     *   }
     * }
     */
    public boolean deleteUser(Long userId) {
        if (userId == null) {
            throw new IllegalArgumentException("User ID is required");
        }
        
        User removedUser = users.remove(userId);
        return removedUser != null;
    }
    
    // 数据模型类
    public static class User {
        private Long userId;
        private String email;
        private String name;
        
        public User(Long userId, String email, String name) {
            this.userId = userId;
            this.email = email;
            this.name = name;
        }
        
        // Getters and Setters
        public Long getUserId() { return userId; }
        public void setUserId(Long userId) { this.userId = userId; }
        
        public String getEmail() { return email; }
        public void setEmail(String email) { this.email = email; }
        
        public String getName() { return name; }
        public void setName(String name) { this.name = name; }
    }
    
    public static class CreateUserRequest {
        private String email;
        private String name;
        private String password;
        
        // Getters and Setters
        public String getEmail() { return email; }
        public void setEmail(String email) { this.email = email; }
        
        public String getName() { return name; }
        public void setName(String name) { this.name = name; }
        
        public String getPassword() { return password; }
        public void setPassword(String password) { this.password = password; }
    }
    
    public static class UpdateUserRequest {
        private String email;
        private String name;
        
        // Getters and Setters
        public String getEmail() { return email; }
        public void setEmail(String email) { this.email = email; }
        
        public String getName() { return name; }
        public void setName(String name) { this.name = name; }
    }
}