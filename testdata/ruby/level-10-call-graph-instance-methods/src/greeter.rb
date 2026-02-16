class Greeter
  def greet(name)
    message = self.build_message(name)
    self.format(message)
  end

  def build_message(name)
    "Hello, #{name}!"
  end

  def format(text)
    text.upcase
  end
end
