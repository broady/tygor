/**
 * TygorError is the base class for all tygor client errors.
 * Use instanceof to narrow to RPCError or TransportError.
 */
export class TygorError extends Error {
}
/**
 * RPCError represents an application-level error returned by the tygor server.
 * These have a structured code, message, and optional details.
 */
export class RPCError extends TygorError {
    constructor(code, message, httpStatus, details) {
        super(message);
        this.kind = "rpc";
        this.name = "RPCError";
        this.code = code;
        this.httpStatus = httpStatus;
        this.details = details;
    }
}
/**
 * TransportError represents a transport-level error (proxy, network, malformed response).
 * These occur when the response is not a valid tygor envelope.
 */
export class TransportError extends TygorError {
    constructor(message, httpStatus, rawBody) {
        super(message);
        this.kind = "transport";
        this.name = "TransportError";
        this.httpStatus = httpStatus;
        this.rawBody = rawBody;
    }
}
export function createClient(registry, config) {
    const fetchFn = config.fetch || globalThis.fetch;
    return new Proxy({}, {
        get: (_target, service) => {
            return new Proxy({}, {
                get: (_target, method) => {
                    const opId = `${service}.${method}`;
                    const meta = registry.metadata[opId];
                    if (!meta) {
                        throw new Error(`Unknown RPC method: ${opId}`);
                    }
                    return async (req) => {
                        const headers = config.headers ? config.headers() : {};
                        let url = config.baseUrl + meta.path;
                        const options = {
                            method: meta.method,
                            headers: {
                                ...headers,
                            },
                        };
                        // Methods that use query parameters (no body)
                        const usesQueryParams = meta.method === "GET" || meta.method === "HEAD";
                        if (usesQueryParams) {
                            const params = new URLSearchParams();
                            // Sort keys for consistent URL generation (important for caching)
                            const sortedKeys = Object.keys(req || {}).sort();
                            sortedKeys.forEach((key) => {
                                const value = req[key];
                                if (Array.isArray(value)) {
                                    value.forEach((v) => params.append(key, String(v)));
                                }
                                else if (value !== undefined && value !== null) {
                                    params.append(key, String(value));
                                }
                            });
                            const qs = params.toString();
                            if (qs) {
                                url += "?" + qs;
                            }
                        }
                        else {
                            options.headers = {
                                ...options.headers,
                                "Content-Type": "application/json",
                            };
                            options.body = JSON.stringify(req);
                        }
                        const res = await fetchFn(url, options);
                        const httpStatus = res.status;
                        // Try to parse as JSON
                        let rawBody;
                        let envelope;
                        try {
                            // Clone response so we can read body twice if needed
                            rawBody = await res.clone().text();
                            envelope = JSON.parse(rawBody);
                        }
                        catch {
                            // JSON parse failed - this is a transport error (proxy HTML page, etc.)
                            throw new TransportError(res.statusText || "Failed to parse response", httpStatus, rawBody?.slice(0, 1000) // Truncate for sanity
                            );
                        }
                        // Handle malformed or null envelope
                        if (!envelope || typeof envelope !== "object") {
                            throw new TransportError("Invalid response format", httpStatus, rawBody?.slice(0, 1000));
                        }
                        // Validate envelope has expected structure
                        if (!("result" in envelope) && !("error" in envelope)) {
                            throw new TransportError("Invalid response format: missing result or error field", httpStatus, rawBody?.slice(0, 1000));
                        }
                        // Check for error in envelope - this is an application-level error
                        if (envelope.error) {
                            throw new RPCError(envelope.error.code || "unknown", envelope.error.message || "Unknown error", httpStatus, envelope.error.details);
                        }
                        // Return the unwrapped result
                        return envelope.result;
                    };
                },
            });
        },
    });
}
