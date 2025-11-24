/**
 * TygorError is the base class for all tygor client errors.
 * Use instanceof to narrow to RPCError or TransportError.
 */
export abstract class TygorError extends Error {
  abstract readonly kind: "rpc" | "transport";
}

/**
 * RPCError represents an application-level error returned by the tygor server.
 * These have a structured code, message, and optional details.
 */
export class RPCError extends TygorError {
  readonly kind = "rpc" as const;
  code: string;
  details?: Record<string, any>;
  httpStatus: number;

  constructor(code: string, message: string, httpStatus: number, details?: Record<string, any>) {
    super(message);
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
  readonly kind = "transport" as const;
  httpStatus: number;
  rawBody?: string;

  constructor(message: string, httpStatus: number, rawBody?: string) {
    super(message);
    this.name = "TransportError";
    this.httpStatus = httpStatus;
    this.rawBody = rawBody;
  }
}

/**
 * RPCResponse is the discriminated union type for RPC responses.
 * Use this with the "never throw" client pattern:
 *
 * @example
 * const response = await client.Users.Get.safe({ id });
 * if (response.error) {
 *   console.log(response.error.code);
 * } else {
 *   console.log(response.result.name);
 * }
 */
export type RPCResponse<T> =
  | { result: T; error?: never }
  | { result?: never; error: { code: string; message: string; details?: Record<string, any> } };

/**
 * Empty represents a void response (null).
 */
export type Empty = null;

export type FetchFunction = (url: string, init?: RequestInit) => Promise<Response>;

export interface RPCConfig {
  baseUrl: string;
  headers?: () => Record<string, string>;
  fetch?: FetchFunction;
}

export interface ServiceRegistry<Manifest extends Record<string, any>> {
  manifest: Manifest;
  metadata: Record<string, { method: string; path: string }>;
}

export function createClient<Manifest extends Record<string, any>>(
  registry: ServiceRegistry<Manifest>,
  config: RPCConfig
): Client<Manifest> {
  const fetchFn = config.fetch || globalThis.fetch;

  return new Proxy(
    {},
    {
      get: (_target, service: string) => {
        return new Proxy(
          {},
          {
            get: (_target, method: string) => {
              const opId = `${service}.${method}`;
              const meta = registry.metadata[opId];
              if (!meta) {
                throw new Error(`Unknown RPC method: ${opId}`);
              }

              return async (req: any) => {
                const headers = config.headers ? config.headers() : {};
                let url = config.baseUrl + meta.path;
                const options: RequestInit = {
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
                    } else if (value !== undefined && value !== null) {
                      params.append(key, String(value));
                    }
                  });
                  const qs = params.toString();
                  if (qs) {
                    url += "?" + qs;
                  }
                } else {
                  options.headers = {
                    ...options.headers,
                    "Content-Type": "application/json",
                  };
                  options.body = JSON.stringify(req);
                }

                const res = await fetchFn(url, options);
                const httpStatus = res.status;

                // Try to parse as JSON
                let rawBody: string | undefined;
                let envelope: RPCResponse<any>;
                try {
                  // Clone response so we can read body twice if needed
                  rawBody = await res.clone().text();
                  envelope = JSON.parse(rawBody);
                } catch {
                  // JSON parse failed - this is a transport error (proxy HTML page, etc.)
                  throw new TransportError(
                    res.statusText || "Failed to parse response",
                    httpStatus,
                    rawBody?.slice(0, 1000) // Truncate for sanity
                  );
                }

                // Handle malformed or null envelope
                if (!envelope || typeof envelope !== "object") {
                  throw new TransportError(
                    "Invalid response format",
                    httpStatus,
                    rawBody?.slice(0, 1000)
                  );
                }

                // Validate envelope has expected structure
                if (!("result" in envelope) && !("error" in envelope)) {
                  throw new TransportError(
                    "Invalid response format: missing result or error field",
                    httpStatus,
                    rawBody?.slice(0, 1000)
                  );
                }

                // Check for error in envelope - this is an application-level error
                if (envelope.error) {
                  throw new RPCError(
                    envelope.error.code || "unknown",
                    envelope.error.message || "Unknown error",
                    httpStatus,
                    envelope.error.details
                  );
                }

                // Return the unwrapped result
                return envelope.result;
              };
            },
          }
        );
      },
    }
  ) as Client<Manifest>;
}

// Type helpers to transform the flat Manifest into a nested Client structure
type ServiceName<K> = K extends `${infer S}.${string}` ? S : never;
type MethodName<K> = K extends `${string}.${infer M}` ? M : never;

type ServiceMethods<M, S extends string> = {
  [K in keyof M as K extends `${S}.${infer Method}` ? Method : never]: M[K] extends {
    req: infer Req;
    res: infer Res;
  }
    ? (req: Req) => Promise<Res>
    : never;
};

export type Client<M> = {
  [K in keyof M as ServiceName<K>]: ServiceMethods<M, ServiceName<K>>;
};
