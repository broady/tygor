// Status type is now generated: import { GetStatusResponse } from "../devserver/types"

/** Type reference in the IR schema */
export interface IRTypeRef {
  kind: string;
  name?: string;
  primitiveKind?: string;
  element?: IRTypeRef;
}

/** Field descriptor in a struct type */
export interface IRField {
  name: string;
  jsonName: string;
}

/** Type descriptor (struct, etc.) */
export interface IRType {
  kind: string;
  Name: { name: string; package: string };
  Fields?: IRField[];
}

/** Discovery schema from discovery.json */
export interface DiscoverySchema {
  Types?: IRType[];
  Services?: Array<{
    name: string;
    endpoints: Array<{
      name: string;
      httpMethod?: string;
      request?: IRTypeRef;
      response?: IRTypeRef;
    }>;
  }>;
}

/** RPC error reported by @tygor/client */
export interface TygorRpcError {
  service: string;
  method: string;
  code: string;
  message: string;
  timestamp: number;
}
