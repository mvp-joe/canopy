class Store
  def initialize
    @data = {}
    @next_id = 1
  end

  def next_id
    id = @next_id
    @next_id += 1
    id
  end

  def save(record)
    @data[record[:id]] = record
  end

  def find(id)
    @data[id]
  end

  def update(record)
    @data[record[:id]] = record
  end

  def all
    @data.values
  end

  def delete(id)
    @data.delete(id)
  end

  def count
    @data.size
  end
end
