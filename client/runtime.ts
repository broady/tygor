/**
 * TygorError is the base class for all tygor client errors.
 * Use instanceof to narrow to ServerError or TransportError.
 */
export abstract class TygorError extends Error {
  abstract readonly kind: "server" | "transport";

  constructor(message: string, options?: ErrorOptions) {
    super(message, options);
  }
}

/**
 * ServerError represents an application-level error returned by the tygor server.
 * These have a structured code, message, and optional details.
 */
export class ServerError extends TygorError {
  readonly kind = "server" as const;
  code: string;
  details?: Record<string, any>;
  httpStatus: number;

  constructor(code: string, message: string, httpStatus: number, details?: Record<string, any>) {
    super(message);
    this.name = "ServerError";
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

  constructor(message: string, httpStatus: number, cause?: unknown, rawBody?: string) {
    super(message, { cause });
    this.name = "TransportError";
    this.httpStatus = httpStatus;
    this.rawBody = rawBody;
  }
}

/**
 * Response is the discriminated union type for service responses.
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
export type Response<T> =
  | { result: T; error?: never }
  | { result?: never; error: { code: string; message: string; details?: Record<string, any> } };

/**
 * Empty represents a void response (null).
 */
export type Empty = null;

export type FetchFunction = (url: string, init?: RequestInit) => Promise<globalThis.Response>;

/**
 * Standard Schema interface (v1) for runtime validation.
 * Compatible with Zod, Valibot, ArkType, and other conforming libraries.
 * @see https://standardschema.dev/
 */
export interface StandardSchema<Input = unknown, Output = Input> {
  readonly "~standard": {
    readonly version: 1;
    readonly vendor: string;
    readonly validate: (value: unknown) => StandardSchemaResult<Output> | Promise<StandardSchemaResult<Output>>;
  };
}

/** Result of Standard Schema validation */
export type StandardSchemaResult<T> = StandardSchemaSuccess<T> | StandardSchemaFailure;

/** Successful validation result */
export interface StandardSchemaSuccess<T> {
  readonly value: T;
  readonly issues?: undefined;
}

/** Failed validation result */
export interface StandardSchemaFailure {
  readonly issues: readonly StandardSchemaIssue[];
}

/** Validation issue from Standard Schema */
export interface StandardSchemaIssue {
  readonly message: string;
  readonly path?: readonly (PropertyKey | { readonly key: PropertyKey })[];
}

/**
 * ValidationError is thrown when client-side schema validation fails.
 */
export class ValidationError extends Error {
  readonly kind = "validation" as const;
  issues: readonly StandardSchemaIssue[];
  direction: "request" | "response";
  endpoint: string;

  constructor(endpoint: string, direction: "request" | "response", issues: readonly StandardSchemaIssue[]) {
    const paths = issues.map((i) => i.path?.join(".") || "(root)").join(", ");
    super(`${direction} validation failed for ${endpoint}: ${paths}`);
    this.name = "ValidationError";
    this.endpoint = endpoint;
    this.direction = direction;
    this.issues = issues;
  }
}

/**
 * Schema map entry for a single endpoint.
 */
export interface SchemaMapEntry {
  request: StandardSchema;
  response: StandardSchema;
}

/**
 * Schema map type - maps endpoint names to their request/response schemas.
 */
export type SchemaMap = Record<string, SchemaMapEntry>;

/**
 * Validation configuration.
 */
export interface ValidateConfig {
  /** Validate request data before sending. Default: true if schemas provided */
  request?: boolean;
  /** Validate response data after receiving. Default: false */
  response?: boolean;
}

export interface ClientConfig {
  baseUrl?: string;
  headers?: () => Record<string, string>;
  fetch?: FetchFunction;
  /** Schema map for client-side validation. Import from schemas.map.ts */
  schemas?: SchemaMap;
  /** Validation options. Only applies if schemas is provided */
  validate?: ValidateConfig;
}

/**
 * Emit a custom event for tygor devtools to display RPC errors.
 * Only emits in browser environment.
 */
function emitRpcError(service: string, method: string, code: string, message: string): void {
  if (typeof window !== "undefined" && typeof CustomEvent !== "undefined") {
    window.dispatchEvent(
      new CustomEvent("tygor:rpc-error", {
        detail: { service, method, code, message, timestamp: Date.now() },
      })
    );
  }
}

export interface ServiceRegistry<Manifest extends Record<string, any>> {
  manifest: Manifest;
  metadata: Record<string, { path: string; primitive: "query" | "exec" | "stream" }>;
}

/**
 * Options for streaming requests.
 */
export interface StreamOptions {
  /** AbortSignal to cancel the stream */
  signal?: AbortSignal;
}

/**
 * Stream represents a server-sent event stream.
 * It's both an AsyncIterable (for `for await` loops) and has a subscribe method
 * for reactive frameworks like React, Svelte, etc.
 */
export interface Stream<T> extends AsyncIterable<T> {
  /**
   * Subscribe to stream values. Returns an unsubscribe function.
   * Designed to work naturally with React's useEffect:
   *
   * @example
   * useEffect(() => client.System.InfoStream({}).subscribe(setInfo), []);
   *
   * @param onValue - Called for each value emitted by the stream
   * @param onError - Optional error handler. AbortErrors from cleanup are automatically ignored.
   * @returns Unsubscribe function that cancels the stream
   */
  subscribe(onValue: (value: T) => void, onError?: (error: Error) => void): () => void;
}

export function createClient<Manifest extends Record<string, any>>(
  registry: ServiceRegistry<Manifest>,
  config: ClientConfig = {}
): Client<Manifest> {
  const fetchFn = config.fetch || globalThis.fetch;

  // Determine validation settings
  const schemas = config.schemas;
  const validateRequest = schemas && (config.validate?.request ?? true); // Default: true if schemas provided
  const validateResponse = schemas && (config.validate?.response ?? false); // Default: false

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
                throw new Error(`Unknown service method: ${opId}`);
              }

              // Streaming endpoints return AsyncIterable
              if (meta.primitive === "stream") {
                return (req: any, options?: StreamOptions) => {
                  return createSSEStream(
                    opId,
                    service,
                    method,
                    meta,
                    req,
                    options,
                    config,
                    fetchFn,
                    schemas,
                    validateRequest,
                    validateResponse
                  );
                };
              }

              // Unary endpoints return Promise
              return async (req: any) => {
                // Request validation (before sending)
                if (validateRequest && schemas?.[opId]?.request) {
                  const schema = schemas[opId].request;
                  const result = await schema["~standard"].validate(req);
                  if (result.issues) {
                    const err = new ValidationError(opId, "request", result.issues);
                    emitRpcError(service, method, "validation_error", err.message);
                    throw err;
                  }
                }

                const headers = config.headers ? config.headers() : {};
                let url = (config.baseUrl || "") + meta.path;
                const httpMethod = meta.primitive === "query" ? "GET" : "POST";
                const options: RequestInit = {
                  method: httpMethod,
                  headers: {
                    ...headers,
                  },
                };

                // Query primitive uses query parameters (no body)
                const usesQueryParams = meta.primitive === "query";

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

                let res: globalThis.Response;
                try {
                  res = await fetchFn(url, options);
                } catch (e) {
                  // Network error (server down, CORS, DNS failure, etc.)
                  const msg = e instanceof Error ? e.message : "Network error";
                  emitRpcError(service, method, "network_error", msg);
                  throw new TransportError(msg, 0, e);
                }
                const httpStatus = res.status;

                // Try to parse as JSON
                let rawBody = "";
                let envelope: Response<any>;
                try {
                  // Clone response so we can read body twice if needed
                  rawBody = await res.clone().text();
                  envelope = JSON.parse(rawBody);
                } catch {
                  // JSON parse failed - this is a transport error (proxy HTML page, etc.)
                  const msg = res.statusText || "Failed to parse response";
                  emitRpcError(service, method, "transport_error", msg);
                  throw new TransportError(msg, httpStatus, undefined, rawBody.slice(0, 1000));
                }

                // Handle malformed or null envelope
                if (!envelope || typeof envelope !== "object") {
                  emitRpcError(service, method, "transport_error", "Invalid response format");
                  throw new TransportError("Invalid response format", httpStatus, undefined, rawBody.slice(0, 1000));
                }

                // Validate envelope has expected structure
                if (!("result" in envelope) && !("error" in envelope)) {
                  const msg = "Invalid response format: missing result or error field";
                  emitRpcError(service, method, "transport_error", msg);
                  throw new TransportError(msg, httpStatus, undefined, rawBody.slice(0, 1000));
                }

                // Check for error in envelope - this is an application-level error
                if (envelope.error) {
                  const code = envelope.error.code || "unknown";
                  const msg = envelope.error.message || "Unknown error";
                  emitRpcError(service, method, code, msg);
                  throw new ServerError(code, msg, httpStatus, envelope.error.details);
                }

                // Response validation (after receiving)
                if (validateResponse && schemas?.[opId]?.response) {
                  const schema = schemas[opId].response;
                  const result = await schema["~standard"].validate(envelope.result);
                  if (result.issues) {
                    const err = new ValidationError(opId, "response", result.issues);
                    emitRpcError(service, method, "validation_error", err.message);
                    throw err;
                  }
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

/**
 * Creates a Stream that emits SSE events from the server.
 * The returned object is both an AsyncIterable and has a subscribe() method.
 */
function createSSEStream<T>(
  opId: string,
  service: string,
  method: string,
  meta: { path: string; primitive: "query" | "exec" | "stream" },
  req: any,
  options: StreamOptions | undefined,
  config: ClientConfig,
  fetchFn: FetchFunction,
  schemas: SchemaMap | undefined,
  validateRequest: boolean | undefined,
  validateResponse: boolean | undefined
): Stream<T> {
  // Factory function to create the async iterator - called fresh for each iteration/subscription
  const createIterator = (signal?: AbortSignal): AsyncIterator<T> => {
      let reader: ReadableStreamDefaultReader<Uint8Array> | null = null;
      let buffer = "";
      let done = false;
      let httpStatus = 0;

      return {
        async next(): Promise<IteratorResult<T>> {
          // First call: establish connection
          if (!reader) {
            // Request validation (before sending)
            if (validateRequest && schemas?.[opId]?.request) {
              const schema = schemas[opId].request;
              const result = await schema["~standard"].validate(req);
              if (result.issues) {
                const err = new ValidationError(opId, "request", result.issues);
                emitRpcError(service, method, "validation_error", err.message);
                throw err;
              }
            }

            const headers = config.headers ? config.headers() : {};
            const url = (config.baseUrl || "") + meta.path;
            const fetchOptions: RequestInit = {
              method: "POST", // stream primitive always uses POST
              headers: {
                ...headers,
                "Content-Type": "application/json",
                Accept: "text/event-stream",
              },
              body: JSON.stringify(req),
              signal,
            };

            let res: globalThis.Response;
            try {
              res = await fetchFn(url, fetchOptions);
            } catch (e) {
              const msg = e instanceof Error ? e.message : "Network error";
              emitRpcError(service, method, "network_error", msg);
              throw new TransportError(msg, 0, e);
            }

            httpStatus = res.status;

            // Check for non-SSE error response (validation, auth, etc.)
            const contentType = res.headers.get("Content-Type") || "";
            if (!contentType.includes("text/event-stream")) {
              // Try to parse as JSON error
              let rawBody = "";
              try {
                rawBody = await res.text();
                const envelope = JSON.parse(rawBody);
                if (envelope.error) {
                  const code = envelope.error.code || "unknown";
                  const msg = envelope.error.message || "Unknown error";
                  emitRpcError(service, method, code, msg);
                  throw new ServerError(code, msg, httpStatus, envelope.error.details);
                }
              } catch (e) {
                if (e instanceof ServerError) throw e;
                const msg = res.statusText || "Failed to establish stream";
                emitRpcError(service, method, "transport_error", msg);
                throw new TransportError(msg, httpStatus, undefined, rawBody.slice(0, 1000));
              }
            }

            if (!res.body) {
              throw new TransportError("Response body is null", httpStatus);
            }

            reader = res.body.getReader();
          }

          if (done) {
            return { done: true, value: undefined as any };
          }

          const decoder = new TextDecoder();

          // Read and parse SSE events
          while (true) {
            // Check for complete events in buffer
            const eventEnd = buffer.indexOf("\n\n");
            if (eventEnd !== -1) {
              const eventText = buffer.slice(0, eventEnd);
              buffer = buffer.slice(eventEnd + 2);

              // Parse SSE event
              const lines = eventText.split("\n");
              for (const line of lines) {
                if (line.startsWith("data: ")) {
                  const data = line.slice(6);
                  try {
                    const envelope = JSON.parse(data) as Response<T>;

                    // Check for error event
                    if (envelope.error) {
                      done = true;
                      const code = envelope.error.code || "unknown";
                      const msg = envelope.error.message || "Unknown error";
                      emitRpcError(service, method, code, msg);
                      throw new ServerError(code, msg, httpStatus, envelope.error.details);
                    }

                    // Response validation
                    if (validateResponse && schemas?.[opId]?.response) {
                      const schema = schemas[opId].response;
                      const result = await schema["~standard"].validate(envelope.result);
                      if (result.issues) {
                        const err = new ValidationError(opId, "response", result.issues);
                        emitRpcError(service, method, "validation_error", err.message);
                        throw err;
                      }
                    }

                    return { done: false, value: envelope.result as T };
                  } catch (e) {
                    if (e instanceof ServerError || e instanceof ValidationError) throw e;
                    emitRpcError(service, method, "transport_error", "Failed to parse SSE event");
                    throw new TransportError("Failed to parse SSE event", httpStatus, e, data);
                  }
                }
              }
            }

            // Read more data
            const { value, done: streamDone } = await reader.read();
            if (streamDone) {
              done = true;
              return { done: true, value: undefined as any };
            }

            buffer += decoder.decode(value, { stream: true });
          }
        },

        async return(): Promise<IteratorResult<T>> {
          done = true;
          if (reader) {
            await reader.cancel();
            reader = null;
          }
          return { done: true, value: undefined as any };
        },
      };
  };

  return {
    [Symbol.asyncIterator](): AsyncIterator<T> {
      // Use the signal from options if provided (for manual for-await usage)
      return createIterator(options?.signal);
    },

    subscribe(onValue: (value: T) => void, onError?: (error: Error) => void): () => void {
      const controller = new AbortController();
      const iterator = createIterator(controller.signal);

      (async () => {
        while (true) {
          const { done, value } = await iterator.next();
          if (done) break;
          onValue(value);
        }
      })().catch((err) => {
        // Silently ignore AbortError from cleanup - this is expected behavior
        if (err.name === "AbortError") return;
        if (onError) {
          onError(err);
        } else {
          console.error("Stream error:", err);
        }
      });

      return () => controller.abort();
    },
  };
}

// Type helpers to transform the flat Manifest into a nested Client structure
type ServiceName<K> = K extends `${infer S}.${string}` ? S : never;

type ServiceMethods<M, S extends string> = {
  [K in keyof M as K extends `${S}.${infer Method}` ? Method : never]: M[K] extends {
    req: infer Req;
    res: infer Res;
    primitive: "stream";
  }
    ? (req: Req, options?: StreamOptions) => Stream<Res>
    : M[K] extends {
          req: infer Req;
          res: infer Res;
        }
      ? (req: Req) => Promise<Res>
      : never;
};

export type Client<M> = {
  [K in keyof M as ServiceName<K>]: ServiceMethods<M, ServiceName<K>>;
};
