/**
 * TygorError is the base class for all tygor client errors.
 * Use instanceof to narrow to ServerError or TransportError.
 */
export abstract class TygorError extends Error {
  abstract readonly kind: "server" | "transport";
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

  constructor(message: string, httpStatus: number, rawBody?: string) {
    super(message);
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
  baseUrl: string;
  headers?: () => Record<string, string>;
  fetch?: FetchFunction;
  /** Schema map for client-side validation. Import from schemas.map.ts */
  schemas?: SchemaMap;
  /** Validation options. Only applies if schemas is provided */
  validate?: ValidateConfig;
}

export interface ServiceRegistry<Manifest extends Record<string, any>> {
  manifest: Manifest;
  metadata: Record<string, { method: string; path: string }>;
}

export function createClient<Manifest extends Record<string, any>>(
  registry: ServiceRegistry<Manifest>,
  config: ClientConfig
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

              return async (req: any) => {
                // Request validation (before sending)
                if (validateRequest && schemas?.[opId]?.request) {
                  const schema = schemas[opId].request;
                  const result = await schema["~standard"].validate(req);
                  if (result.issues) {
                    throw new ValidationError(opId, "request", result.issues);
                  }
                }

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
                let rawBody = "";
                let envelope: Response<any>;
                try {
                  // Clone response so we can read body twice if needed
                  rawBody = await res.clone().text();
                  envelope = JSON.parse(rawBody);
                } catch {
                  // JSON parse failed - this is a transport error (proxy HTML page, etc.)
                  throw new TransportError(
                    res.statusText || "Failed to parse response",
                    httpStatus,
                    rawBody.slice(0, 1000) // Truncate for sanity
                  );
                }

                // Handle malformed or null envelope
                if (!envelope || typeof envelope !== "object") {
                  throw new TransportError(
                    "Invalid response format",
                    httpStatus,
                    rawBody.slice(0, 1000)
                  );
                }

                // Validate envelope has expected structure
                if (!("result" in envelope) && !("error" in envelope)) {
                  throw new TransportError(
                    "Invalid response format: missing result or error field",
                    httpStatus,
                    rawBody.slice(0, 1000)
                  );
                }

                // Check for error in envelope - this is an application-level error
                if (envelope.error) {
                  throw new ServerError(
                    envelope.error.code || "unknown",
                    envelope.error.message || "Unknown error",
                    httpStatus,
                    envelope.error.details
                  );
                }

                // Response validation (after receiving)
                if (validateResponse && schemas?.[opId]?.response) {
                  const schema = schemas[opId].response;
                  const result = await schema["~standard"].validate(envelope.result);
                  if (result.issues) {
                    throw new ValidationError(opId, "response", result.issues);
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
