package com.example.api;

import com.flowspec.annotations.ServiceSpec;

/**
 * Health check service
 */
public class HealthService {

    /**
     * @ServiceSpec
     * operationId: "healthCheck"
     * description: "Health check endpoint"
     * preconditions: {}
     * postconditions: {
     *   "success_status": {"==": [{"var": "span.attributes.http.status_code"}, 200]},
     *   "status_returned": {"!=": [{"var": "span.attributes.response.body.status"}, null]}
     * }
     */
    public HealthStatus getHealth() {
        // Implementation
        return null;
    }
}
