export type Manifest = {
  title: string;
  description?: string;
  requirements?: Requirement[];
  root?: CommandRef[];
  commands: CommandSpec[];
};

type Requirement = {
  name: string;
  link?: string;
};

type CommandRef = {
  command: string;
  title: string;
  description?: string;
  params?: Record<string, string | number | boolean>;
};

export type CommandSpec = {
  name: string;
  title: string;
  mode: "list" | "detail" | "tty" | "silent";
  hidden?: boolean;
  description?: string;
  params?: CommandParam[];
};

type CommandParam = StringParam | BooleanParam | NumberParam;

type ParamsProps = {
  name: string;
  description?: string;
  required?: boolean;
};

type StringParam = {
  type: "string";
  default?: string;
} & ParamsProps;

type BooleanParam = {
  type: "boolean";
  default?: boolean;
} & ParamsProps;

type NumberParam = {
  type: "number";
  default?: number;
} & ParamsProps;

type PayloadParams = Record<string, string | number | boolean>;
export type Payload<T extends PayloadParams = PayloadParams> = {
  command: string;
  params: T;
  query?: string;
  cwd?: string;
};
