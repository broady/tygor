/**
 * TygorError is the base class for all tygor client errors.
 * Use instanceof to narrow to RPCError or TransportError.
 */
export declare abstract class TygorError extends Error {
    abstract readonly kind: "rpc" | "transport";
}
/**
 * RPCError represents an application-level error returned by the tygor server.
 * These have a structured code, message, and optional details.
 */
export declare class RPCError extends TygorError {
    readonly kind: "rpc";
    code: string;
    details?: Record<string, any>;
    httpStatus: number;
    constructor(code: string, message: string, httpStatus: number, details?: Record<string, any>);
}
/**
 * TransportError represents a transport-level error (proxy, network, malformed response).
 * These occur when the response is not a valid tygor envelope.
 */
export declare class TransportError extends TygorError {
    readonly kind: "transport";
    httpStatus: number;
    rawBody?: string;
    constructor(message: string, httpStatus: number, rawBody?: string);
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
export type RPCResponse<T> = {
    result: T;
    error?: never;
} | {
    result?: never;
    error: {
        code: string;
        message: string;
        details?: Record<string, any>;
    };
};
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
    metadata: Record<string, {
        method: string;
        path: string;
    }>;
}
export declare function createClient<Manifest extends Record<string, any>>(registry: ServiceRegistry<Manifest>, config: RPCConfig): Client<Manifest>;
type ServiceName<K> = K extends `${infer S}.${string}` ? S : never;
type ServiceMethods<M, S extends string> = {
    [K in keyof M as K extends `${S}.${infer Method}` ? Method : never]: M[K] extends {
        req: infer Req;
        res: infer Res;
    } ? (req: Req) => Promise<Res> : never;
};
export type Client<M> = {
    [K in keyof M as ServiceName<K>]: ServiceMethods<M, ServiceName<K>>;
};
export {};
//# sourceMappingURL=runtime.d.ts.map