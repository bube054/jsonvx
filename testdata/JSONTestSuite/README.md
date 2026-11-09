# JSONTestSuite

The `.json` files in this folder are the `test_parsing` cases from
[JSONTestSuite](https://github.com/nst/JSONTestSuite) by Nicolas Seriot,
commit 1ef36fa (2024-11-22), included under its MIT license (see `LICENSE`).

File names say what a strict JSON parser must do:

- `y_*.json`: must be accepted
- `n_*.json`: must be rejected
- `i_*.json`: either is allowed

They are used by `TestJSONTestSuite` in `jsontestsuite_test.go`.
`.gitattributes` stores them byte for byte, so line endings are never converted.
