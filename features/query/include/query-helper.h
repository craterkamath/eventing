// Copyright (c) 2019 Couchbase, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
// or implied. See the License for the specific language governing
// permissions and limitations under the License.

#ifndef QUERY_HELPER_H
#define QUERY_HELPER_H

#include <string>
#include <unordered_map>
#include <utility>
#include <v8.h>
#include <vector>

#include "info.h"
#include "isolate_data.h"
#include "query-info.h"
#include "query-row.h"

namespace Query {
class Helper {
public:
  Helper(v8::Isolate *isolate, const v8::Local<v8::Context> &context);
  ~Helper();

  Helper(const Helper &) = delete;
  Helper(Helper &&) = delete;
  Helper &operator=(const Helper &) = delete;
  Helper &operator=(Helper &&) = delete;

  static ::Info ValidateQuery(const v8::FunctionCallbackInfo<v8::Value> &args);
  Query::Info CreateQuery(const v8::FunctionCallbackInfo<v8::Value> &args);
  ::Info AccountLCBError(const std::string &err_str);
  void AccountLCBError(int err_code);
  void HandleRowError(const Query::Row &row);
  std::string ErrorFormat(const std::string &message, lcb_t connection,
                          lcb_error_t error);
  static int GetConsistency(const std::string &consistency);

private:
  struct ErrorCodesInfo : public ::Info {
    ErrorCodesInfo(const ::Info &info) : ::Info(info.is_fatal, info.msg) {}
    ErrorCodesInfo(bool is_fatal, const std::string &msg)
        : ::Info(is_fatal, msg) {}
    ErrorCodesInfo(std::vector<int64_t> &errors) : ::Info(false) {
      std::swap(this->err_codes, errors);
    }

    std::vector<int64_t> err_codes;
  };

  struct NamedParamsInfo : public ::Info {
    NamedParamsInfo(const bool is_fatal, const std::string &msg)
        : ::Info(is_fatal, msg) {}
    NamedParamsInfo(std::unordered_map<std::string, std::string> &named_params)
        : ::Info(false) {
      std::swap(this->named_params, named_params);
    }

    std::unordered_map<std::string, std::string> named_params;
  };

  struct PosParamsInfo : public ::Info {
    PosParamsInfo(const bool is_fatal, const std::string &msg)
        : ::Info(is_fatal, msg) {}
    PosParamsInfo(std::vector<std::string> &pos_params) : ::Info(false) {
      std::swap(this->pos_params, pos_params);
    }

    std::vector<std::string> pos_params;
  };

  ErrorCodesInfo GetErrorCodes(const std::string &error);
  ErrorCodesInfo GetErrorCodes(const v8::Local<v8::Value> &errors_val);
  NamedParamsInfo GetNamedParams(const v8::Local<v8::Value> &arg) const;
  PosParamsInfo GetPosParams(const v8::Local<v8::Value> &arg) const;

  v8::Isolate *isolate_;
  v8::Persistent<v8::Context> context_;
  Options::Extractor opt_extractor_;
};
} // namespace Query

void AddLcbException(const IsolateData *isolate_data, int code);

#endif
