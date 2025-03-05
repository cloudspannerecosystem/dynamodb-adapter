# Limitations of ExecuteStatement API in DynamoDB Proxy Adapter

This document outlines the known limitations of the `ExecuteStatement` API when using the DynamoDB Proxy Adapter. Understanding these limitations is critical for effective use of the API in your applications.

## Limitations

1. **Does Not Support OR Operator**
   The `ExecuteStatement` API does not accommodate queries that utilize the OR operator. Queries must be constructed using AND conditions only.

2. **Does Not Support Complex Conditions in WHERE Clause**
   Complex conditional statements involving multiple attributes that require logical operations are not supported. Only simple equality checks can be used in the WHERE clause.

3. **Does Not Support WHERE NOT**
   The API does not support negation in the WHERE clause. Queries cannot filter out results using NOT conditions.

4. **Does Not Support IN Operator**
   The IN operator for checking multiple values against an attribute is not supported. Queries must use multiple equality checks instead.

5. **Does Not Support Map, List, and Set Datatypes**
   The API currently does not support the usage of Map, List, or Set datatypes. All attribute values must be either scalar types such as string, number, or boolean.

6. **Does Not Support COUNT Function**
   The API does not support aggregate functions such as COUNT. Queries that require aggregation must be handled differently.

7. **NextToken is Not Supported**
   The API does not support pagination using a NextToken mechanism, limiting the ability to fetch large sets of results incrementally.

8. **Supports Comparison Operators (> and <)**
   The API supports greater than (`>`) and less than (`<`) comparison operators for numeric and string types. This allows for range queries to some extent.

## Conclusion

When using the `ExecuteStatement` API with the DynamoDB Proxy Adapter, be aware of these limitations to design your queries accordingly.
