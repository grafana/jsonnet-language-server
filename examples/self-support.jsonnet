{
  // Support of self within the same structure
  my_attribute: 'value',
  my_other_attribute: self.my_attribute,
}
+ {
  // Support of self in merged structures
  from_merge: self.my_attribute,
}
+ {
  // Support of self as a local reference
  local this = self,
  from_local: this.my_attribute,
}
