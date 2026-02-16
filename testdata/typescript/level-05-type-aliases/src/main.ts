type ID = string;

type Result = {
  value: string;
  error: string | null;
};

type Status = "active" | "inactive" | "pending";

function createId(name: string): ID {
  return name + "-001";
}

const current: Status = "active";
