module Greetable
  def greet
    "hello"
  end
end

module Farewell
  def goodbye
    "bye"
  end
end

class Person
  include Greetable

  def initialize(name)
    @name = name
  end
end
