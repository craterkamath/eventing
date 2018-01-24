// Copyright (c) 2017 Couchbase, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
// or implied. See the License for the specific language governing
// permissions and limitations under the License.

#include "comm.h"
#include "log.h"
#include "utils.h"

CURLClient::CURLClient() : headers(nullptr) { curl_handle = curl_easy_init(); }

CURLClient::~CURLClient() { curl_easy_cleanup(curl_handle); }

size_t CURLClient::BodyCallback(void *buffer, size_t size, size_t nmemb,
                                void *cookie) {
  auto realsize = size * nmemb;
  auto data = static_cast<std::string *>(cookie);
  auto content = static_cast<char *>(buffer);
  data->assign(&content[0], &content[0] + realsize);
  return realsize;
}

size_t CURLClient::HeaderCallback(char *buffer, size_t size, size_t nitems,
                                  void *cookie) {
  auto realsize = size * nitems;
  auto headers =
      static_cast<std::unordered_map<std::string, std::string> *>(cookie);
  auto header = std::string(static_cast<char *>(buffer));

  // Split the header into key:value
  auto find = header.find(':');
  if (find != std::string::npos) {
    (*headers)[header.substr(0, find)] =
        header.substr(find + 1); // Adding 1 to discount the ':'
  }

  return realsize;
}

CURLResponse CURLClient::HTTPPost(const std::vector<std::string> &header_list,
                                  const std::string &url,
                                  const std::string &body) {
  CURLResponse response;

  code = curl_easy_setopt(curl_handle, CURLOPT_URL, url.c_str());
  if (code != CURLE_OK) {
    response.is_error = true;
    response.response =
        "Unable to set URL: " + std::string(curl_easy_strerror(code));
    return response;
  }

  for (const auto &header : header_list) {
    headers = curl_slist_append(headers, header.c_str());
  }

  code = curl_easy_setopt(curl_handle, CURLOPT_HTTPHEADER, headers);
  if (code != CURLE_OK) {
    response.is_error = true;
    response.response = "Unable to do set HTTP header(s): " +
                        std::string(curl_easy_strerror(code));
    return response;
  }

  code = curl_easy_setopt(curl_handle, CURLOPT_POSTFIELDS, body.c_str());
  if (code != CURLE_OK) {
    response.is_error = true;
    response.response =
        "Unable to set POST body: " + std::string(curl_easy_strerror(code));
    return response;
  }

  code = curl_easy_setopt(curl_handle, CURLOPT_WRITEFUNCTION,
                          CURLClient::BodyCallback);
  if (code != CURLE_OK) {
    response.is_error = true;
    response.response = "Unable to set body callback function: " +
                        std::string(curl_easy_strerror(code));
    return response;
  }

  code = curl_easy_setopt(curl_handle, CURLOPT_HEADERFUNCTION,
                          CURLClient::HeaderCallback);
  if (code != CURLE_OK) {
    response.is_error = true;
    response.response = "Unable to set header callback function: " +
                        std::string(curl_easy_strerror(code));
    return response;
  }

  code = curl_easy_setopt(curl_handle, CURLOPT_HEADERDATA,
                          (void *)&response.headers);
  if (code != CURLE_OK) {
    response.is_error = true;
    response.response = "Unable to set cookie for headers: " +
                        std::string(curl_easy_strerror(code));
    return response;
  }

  code = curl_easy_setopt(curl_handle, CURLOPT_WRITEDATA,
                          (void *)&response.response);
  if (code != CURLE_OK) {
    response.is_error = true;
    response.response = "Unable to set cookie for body: " +
                        std::string(curl_easy_strerror(code));
    return response;
  }

  code = curl_easy_setopt(curl_handle, CURLOPT_USERAGENT, "libcurl-agent/1.0");
  if (code != CURLE_OK) {
    response.is_error = true;
    response.response =
        "Unable to set user agent: " + std::string(curl_easy_strerror(code));
    return response;
  }

  code = curl_easy_setopt(curl_handle, CURLOPT_TIMEOUT, 30L);
  if (code != CURLE_OK) {
    response.is_error = true;
    response.response = "Unable to set timeout";
    return response;
  }

  code = curl_easy_perform(curl_handle);
  if (code != CURLE_OK) {
    response.is_error = true;
    response.response =
        "Unable to do HTTP POST: " + std::string(curl_easy_strerror(code));
    return response;
  }

  response.is_error = false;
  return response;
}

Communicator::Communicator(const std::string &host_port, v8::Isolate *isolate)
    : isolate(isolate) {
  get_creds_url = "http://localhost:" + host_port + "/getCreds";
}

CredsInfo Communicator::GetCreds(const std::string &endpoint) {
  v8::HandleScope handle_scope(isolate);

  auto context = v8::Context::New(isolate);
  v8::Context::Scope context_scope(context);

  CURLClient curl;
  auto response =
      curl.HTTPPost({"Content-Type: text/plain"}, get_creds_url, endpoint);

  CredsInfo info;
  info.is_error = response.is_error;
  if (response.is_error) {
    info.error = response.response;
    return info;
  }

  if (std::stoi(response.headers["Status"]) != 0) {
    info.is_error = true;
    info.error = response.response;
    return info;
  }

  auto response_obj =
      v8::JSON::Parse(v8Str(isolate, response.response))->ToObject();
  auto username_v8_str = response_obj->Get(v8Str(isolate, "username"));
  auto password_v8_str = response_obj->Get(v8Str(isolate, "password"));
  v8::String::Utf8Value username_utf8(username_v8_str);
  v8::String::Utf8Value password_utf8(password_v8_str);

  info.username = *username_utf8;
  info.password = *password_utf8;
  return info;
}