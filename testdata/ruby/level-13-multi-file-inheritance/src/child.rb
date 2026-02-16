require_relative "base"

class Child < Base
  def process
    self.validate
    Child.create
  end
end
