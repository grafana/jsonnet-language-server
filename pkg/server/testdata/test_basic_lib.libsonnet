{
  greet(name)::
    local greeting = 'Hello, ';
    greeting + name,
  message: self.greet('Zack'),
}
