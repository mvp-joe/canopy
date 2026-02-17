function formatDate(date) {
  return date.toISOString().split('T')[0];
}

function parseConfig(path) {
  return { maxTasks: 100, logLevel: 'info' };
}

function validateTitle(title) {
  if (!title || typeof title !== 'string') return false;
  return title.trim().length > 0 && title.length <= 200;
}

module.exports = { formatDate, parseConfig, validateTitle };
