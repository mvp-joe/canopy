function handleA() {
  const cfg = { run: () => "alpha" };
  return cfg.run();
}

function handleB() {
  const cfg = { run: () => "beta" };
  return cfg.run();
}
