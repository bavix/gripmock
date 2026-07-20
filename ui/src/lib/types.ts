export interface Stub {
  id: string;
  service: string;
  method: string;
  priority: number;
  headers?: StubHeaders;
  input: StubInput;
  inputs?: StubInput[];
  output: StubOutput;
  options?: StubOptions;
  effects?: StubEffect[];
  source?: string;
  session?: string;
  /** Response-only: whether the stub has matched at least once. */
  used?: boolean;
}

export interface StubInput {
  ignoreArrayOrder?: boolean;
  equals?: Record<string, unknown>;
  contains?: Record<string, unknown>;
  matches?: Record<string, unknown>;
  glob?: Record<string, string>;
  anyOf?: StubInputAnyOfElement[];
}

export interface StubInputAnyOfElement {
  ignoreArrayOrder?: boolean;
  equals?: Record<string, unknown>;
  contains?: Record<string, unknown>;
  matches?: Record<string, unknown>;
  glob?: Record<string, string>;
}

export interface StubHeaders {
  equals?: Record<string, string>;
  contains?: Record<string, string>;
  matches?: Record<string, string>;
  anyOf?: StubHeadersAnyOfElement[];
}

export interface StubHeadersAnyOfElement {
  equals?: Record<string, string>;
  contains?: Record<string, string>;
  matches?: Record<string, string>;
}

export interface StubOutput {
  data?: unknown;
  stream?: unknown[];
  headers?: Record<string, string>;
  error?: string;
  code?: number;
  details?: StubOutputDetail[];
  delay?: string;
}

export interface StubOutputDetail {
  type: string;
  [key: string]: unknown;
}

export interface StubOptions {
  times?: number;
}

export interface StubEffect {
  action: 'upsert' | 'delete';
  id?: string;
  stub?: Partial<Stub>;
}

export interface Service {
  id: string;
  package: string;
  name: string;
  methods: Method[];
}

export interface Method {
  id: string;
  name: string;
  methodType: 'unary' | 'client_streaming' | 'server_streaming' | 'bidi_streaming';
  requestType: string;
  responseType: string;
  requestSchema?: ProtoMessageSchema;
  responseSchema?: ProtoMessageSchema;
  clientStreaming: boolean;
  serverStreaming: boolean;
}

export interface ProtoMessageSchema {
  typeName: string;
  recursiveRef?: boolean;
  fields: ProtoFieldSchema[];
}

export interface ProtoFieldSchema {
  name: string;
  jsonName: string;
  number: number;
  kind: string;
  cardinality: 'optional' | 'required' | 'repeated';
  typeName?: string;
  oneof?: string;
  enumValues?: string[];
  map?: boolean;
  mapKeyKind?: string;
  mapValueKind?: string;
  mapValueTypeName?: string;
  message?: ProtoMessageSchema;
}

export interface CallRecord {
  service: string;
  method: string;
  session?: string;
  stubId?: string;
  timestamp: string;
  request?: Record<string, unknown>;
  requests?: Record<string, unknown>[];
  response?: Record<string, unknown>;
  responses?: Record<string, unknown>[];
  code: number;
  error?: string;
  elapsedMs?: number;
}

export interface Dashboard {
  appName: string;
  version: string;
  goVersion: string;
  goos: string;
  goarch: string;
  numCPU: number;
  startedAt: string;
  uptimeSeconds: number;
  ready: boolean;
  historyEnabled: boolean;
  totalServices: number;
  totalStubs: number;
  usedStubs: number;
  unusedStubs: number;
  coveredMethods: number;
  totalMethods: number;
  grpcAddr?: string;
  gatewayAddr?: string;
  httpAddr?: string;
  totalSessions: number;
  runtimeDescriptors: number;
  totalHistory: number;
  historyErrors: number;
}

export interface SearchResponse {
  headers?: Record<string, string>;
  data?: unknown;
  error?: string;
  code?: number;
}

export interface InspectRequest {
  service: string;
  method: string;
  session?: string;
  headers?: Record<string, string>;
  input?: Record<string, unknown>[];
}

export interface InspectReport {
  service: string;
  method: string;
  session?: string;
  matchedStubId?: string;
  similarStubId?: string;
  fallbackToMethod?: boolean;
  error?: string;
  stages?: InspectStage[];
  candidates?: InspectCandidate[];
}

export interface InspectStage {
  name: string;
  before: number;
  after: number;
  removed: number;
}

export interface InspectCandidate {
  id: string;
  service: string;
  method: string;
  session?: string;
  priority: number;
  times: number;
  used: number;
  specificity: number;
  score: number;
  visibleBySession: boolean;
  withinTimes: boolean;
  headersMatched: boolean;
  inputMatched: boolean;
  matched: boolean;
  excludedBy?: string[];
  events?: InspectCandidateEvent[];
}

export interface InspectCandidateEvent {
  stage: string;
  result: string;
  reason: string;
}

export interface VerifyRequest {
  service: string;
  method: string;
  expectedCount: number;
}
