class Processor
  def handle_a
    Formatter.bold("hello")
  end

  def handle_b
    Formatter.italic("world")
  end
end

class Formatter
  def self.bold(text)
    "<b>#{text}</b>"
  end

  def self.italic(text)
    "<i>#{text}</i>"
  end
end
