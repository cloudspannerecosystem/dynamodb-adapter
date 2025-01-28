# Enable all rules by default
all

# Asterisks for unordered lists
rule 'MD004', :style => :asterisk

# Nested lists should e indented with four spaces.
rule 'MD007', :indent => 4

# Allow table and code lines to be longer than 80 chars
rule 'MD013', :ignore_code_blocks => true, :tables => false

# Ordered list item prefixes
rule 'MD029', :style => :ordered

# Spaces after list markers
rule 'MD030', :ul_single => 3, :ul_multi => 3, :ol_single => 2, :ol_multi => 2