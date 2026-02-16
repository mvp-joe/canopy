class Animal
  def speak
    "..."
  end

  def describe
    self.speak
  end
end

class Dog < Animal
  def speak
    "Woof!"
  end

  def greet
    self.describe
  end
end
