module Loggable
  def log(msg)
    puts msg
  end
end

module Serializable
  def to_json
    "{}"
  end
end

class Service
  include Loggable
  include Serializable

  def run
    self.log("starting")
    self.to_json
  end
end
