export class RPCError extends Error {
  code: string;
  details?: Record<string, any>;

  constructor(code: string, message: string, details?: Record<string, any>) {
    super(message);
    this.name = "RPCError";
    this.code = code;
    this.details = details;
  }
}

export interface RPCConfig {
  baseUrl: string;
  headers?: () => Record<string, string>;
}

export function createClient<Manifest extends Record<string, any>>(
  config: RPCConfig,
  metadata: Record<string, { method: string; path: string }>
): Client<Manifest> {
  return new Proxy(
    {},
    {
      get: (_target, service: string) => {
        return new Proxy(
          {},
          {
            get: (_target, method: string) => {
              const opId = `${service}.${method}`;
              const meta = metadata[opId];
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

                const res = await fetch(url, options);
                
                if (!res.ok) {
                  let errorData;
                  try {
                    errorData = await res.json();
                  } catch {
                    throw new RPCError("internal", res.statusText);
                  }
                  // Handle null or malformed error responses
                  if (!errorData || typeof errorData !== 'object') {
                    throw new RPCError("unknown", res.statusText);
                  }
                  throw new RPCError(
                    errorData.code || "unknown",
                    errorData.message || "Unknown error",
                    errorData.details
                  );
                }

                return res.json();
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
