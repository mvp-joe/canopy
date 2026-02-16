function handleA(): string {
  const formatter = (s: string) => s;
  return formatter("success");
}

function handleB(): string {
  const formatter = (s: string) => s;
  return formatter("failure");
}
