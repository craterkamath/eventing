// automatically generated by the FlatBuffers compiler, do not modify


#ifndef FLATBUFFERS_GENERATED_CFGSCHEMA_FLATBUF_CFG_H_
#define FLATBUFFERS_GENERATED_CFGSCHEMA_FLATBUF_CFG_H_

#include "flatbuffers/flatbuffers.h"

namespace flatbuf {
namespace cfg {

struct Config;

struct DepCfg;

struct Bucket;

struct Config FLATBUFFERS_FINAL_CLASS : private flatbuffers::Table {
  enum {
    VT_ID = 4,
    VT_APPCODE = 6,
    VT_APPNAME = 8,
    VT_DEPCFG = 10
  };
  uint32_t id() const {
    return GetField<uint32_t>(VT_ID, 0);
  }
  const flatbuffers::String *appCode() const {
    return GetPointer<const flatbuffers::String *>(VT_APPCODE);
  }
  const flatbuffers::String *appName() const {
    return GetPointer<const flatbuffers::String *>(VT_APPNAME);
  }
  const DepCfg *depCfg() const {
    return GetPointer<const DepCfg *>(VT_DEPCFG);
  }
  bool Verify(flatbuffers::Verifier &verifier) const {
    return VerifyTableStart(verifier) &&
           VerifyField<uint32_t>(verifier, VT_ID) &&
           VerifyField<flatbuffers::uoffset_t>(verifier, VT_APPCODE) &&
           verifier.Verify(appCode()) &&
           VerifyField<flatbuffers::uoffset_t>(verifier, VT_APPNAME) &&
           verifier.Verify(appName()) &&
           VerifyField<flatbuffers::uoffset_t>(verifier, VT_DEPCFG) &&
           verifier.VerifyTable(depCfg()) &&
           verifier.EndTable();
  }
};

struct ConfigBuilder {
  flatbuffers::FlatBufferBuilder &fbb_;
  flatbuffers::uoffset_t start_;
  void add_id(uint32_t id) {
    fbb_.AddElement<uint32_t>(Config::VT_ID, id, 0);
  }
  void add_appCode(flatbuffers::Offset<flatbuffers::String> appCode) {
    fbb_.AddOffset(Config::VT_APPCODE, appCode);
  }
  void add_appName(flatbuffers::Offset<flatbuffers::String> appName) {
    fbb_.AddOffset(Config::VT_APPNAME, appName);
  }
  void add_depCfg(flatbuffers::Offset<DepCfg> depCfg) {
    fbb_.AddOffset(Config::VT_DEPCFG, depCfg);
  }
  ConfigBuilder(flatbuffers::FlatBufferBuilder &_fbb)
        : fbb_(_fbb) {
    start_ = fbb_.StartTable();
  }
  ConfigBuilder &operator=(const ConfigBuilder &);
  flatbuffers::Offset<Config> Finish() {
    const auto end = fbb_.EndTable(start_, 4);
    auto o = flatbuffers::Offset<Config>(end);
    return o;
  }
};

inline flatbuffers::Offset<Config> CreateConfig(
    flatbuffers::FlatBufferBuilder &_fbb,
    uint32_t id = 0,
    flatbuffers::Offset<flatbuffers::String> appCode = 0,
    flatbuffers::Offset<flatbuffers::String> appName = 0,
    flatbuffers::Offset<DepCfg> depCfg = 0) {
  ConfigBuilder builder_(_fbb);
  builder_.add_depCfg(depCfg);
  builder_.add_appName(appName);
  builder_.add_appCode(appCode);
  builder_.add_id(id);
  return builder_.Finish();
}

inline flatbuffers::Offset<Config> CreateConfigDirect(
    flatbuffers::FlatBufferBuilder &_fbb,
    uint32_t id = 0,
    const char *appCode = nullptr,
    const char *appName = nullptr,
    flatbuffers::Offset<DepCfg> depCfg = 0) {
  return CreateConfig(
      _fbb,
      id,
      appCode ? _fbb.CreateString(appCode) : 0,
      appName ? _fbb.CreateString(appName) : 0,
      depCfg);
}

struct DepCfg FLATBUFFERS_FINAL_CLASS : private flatbuffers::Table {
  enum {
    VT_BUCKETS = 4,
    VT_AUTH = 6,
    VT_METADATABUCKET = 8,
    VT_SOURCEBUCKET = 10
  };
  const flatbuffers::Vector<flatbuffers::Offset<Bucket>> *buckets() const {
    return GetPointer<const flatbuffers::Vector<flatbuffers::Offset<Bucket>> *>(VT_BUCKETS);
  }
  const flatbuffers::String *auth() const {
    return GetPointer<const flatbuffers::String *>(VT_AUTH);
  }
  const flatbuffers::String *metadataBucket() const {
    return GetPointer<const flatbuffers::String *>(VT_METADATABUCKET);
  }
  const flatbuffers::String *sourceBucket() const {
    return GetPointer<const flatbuffers::String *>(VT_SOURCEBUCKET);
  }
  bool Verify(flatbuffers::Verifier &verifier) const {
    return VerifyTableStart(verifier) &&
           VerifyField<flatbuffers::uoffset_t>(verifier, VT_BUCKETS) &&
           verifier.Verify(buckets()) &&
           verifier.VerifyVectorOfTables(buckets()) &&
           VerifyField<flatbuffers::uoffset_t>(verifier, VT_AUTH) &&
           verifier.Verify(auth()) &&
           VerifyField<flatbuffers::uoffset_t>(verifier, VT_METADATABUCKET) &&
           verifier.Verify(metadataBucket()) &&
           VerifyField<flatbuffers::uoffset_t>(verifier, VT_SOURCEBUCKET) &&
           verifier.Verify(sourceBucket()) &&
           verifier.EndTable();
  }
};

struct DepCfgBuilder {
  flatbuffers::FlatBufferBuilder &fbb_;
  flatbuffers::uoffset_t start_;
  void add_buckets(flatbuffers::Offset<flatbuffers::Vector<flatbuffers::Offset<Bucket>>> buckets) {
    fbb_.AddOffset(DepCfg::VT_BUCKETS, buckets);
  }
  void add_auth(flatbuffers::Offset<flatbuffers::String> auth) {
    fbb_.AddOffset(DepCfg::VT_AUTH, auth);
  }
  void add_metadataBucket(flatbuffers::Offset<flatbuffers::String> metadataBucket) {
    fbb_.AddOffset(DepCfg::VT_METADATABUCKET, metadataBucket);
  }
  void add_sourceBucket(flatbuffers::Offset<flatbuffers::String> sourceBucket) {
    fbb_.AddOffset(DepCfg::VT_SOURCEBUCKET, sourceBucket);
  }
  DepCfgBuilder(flatbuffers::FlatBufferBuilder &_fbb)
        : fbb_(_fbb) {
    start_ = fbb_.StartTable();
  }
  DepCfgBuilder &operator=(const DepCfgBuilder &);
  flatbuffers::Offset<DepCfg> Finish() {
    const auto end = fbb_.EndTable(start_, 4);
    auto o = flatbuffers::Offset<DepCfg>(end);
    return o;
  }
};

inline flatbuffers::Offset<DepCfg> CreateDepCfg(
    flatbuffers::FlatBufferBuilder &_fbb,
    flatbuffers::Offset<flatbuffers::Vector<flatbuffers::Offset<Bucket>>> buckets = 0,
    flatbuffers::Offset<flatbuffers::String> auth = 0,
    flatbuffers::Offset<flatbuffers::String> metadataBucket = 0,
    flatbuffers::Offset<flatbuffers::String> sourceBucket = 0) {
  DepCfgBuilder builder_(_fbb);
  builder_.add_sourceBucket(sourceBucket);
  builder_.add_metadataBucket(metadataBucket);
  builder_.add_auth(auth);
  builder_.add_buckets(buckets);
  return builder_.Finish();
}

inline flatbuffers::Offset<DepCfg> CreateDepCfgDirect(
    flatbuffers::FlatBufferBuilder &_fbb,
    const std::vector<flatbuffers::Offset<Bucket>> *buckets = nullptr,
    const char *auth = nullptr,
    const char *metadataBucket = nullptr,
    const char *sourceBucket = nullptr) {
  return CreateDepCfg(
      _fbb,
      buckets ? _fbb.CreateVector<flatbuffers::Offset<Bucket>>(*buckets) : 0,
      auth ? _fbb.CreateString(auth) : 0,
      metadataBucket ? _fbb.CreateString(metadataBucket) : 0,
      sourceBucket ? _fbb.CreateString(sourceBucket) : 0);
}

struct Bucket FLATBUFFERS_FINAL_CLASS : private flatbuffers::Table {
  enum {
    VT_BUCKETNAME = 4,
    VT_ALIAS = 6
  };
  const flatbuffers::String *bucketName() const {
    return GetPointer<const flatbuffers::String *>(VT_BUCKETNAME);
  }
  const flatbuffers::String *alias() const {
    return GetPointer<const flatbuffers::String *>(VT_ALIAS);
  }
  bool Verify(flatbuffers::Verifier &verifier) const {
    return VerifyTableStart(verifier) &&
           VerifyField<flatbuffers::uoffset_t>(verifier, VT_BUCKETNAME) &&
           verifier.Verify(bucketName()) &&
           VerifyField<flatbuffers::uoffset_t>(verifier, VT_ALIAS) &&
           verifier.Verify(alias()) &&
           verifier.EndTable();
  }
};

struct BucketBuilder {
  flatbuffers::FlatBufferBuilder &fbb_;
  flatbuffers::uoffset_t start_;
  void add_bucketName(flatbuffers::Offset<flatbuffers::String> bucketName) {
    fbb_.AddOffset(Bucket::VT_BUCKETNAME, bucketName);
  }
  void add_alias(flatbuffers::Offset<flatbuffers::String> alias) {
    fbb_.AddOffset(Bucket::VT_ALIAS, alias);
  }
  BucketBuilder(flatbuffers::FlatBufferBuilder &_fbb)
        : fbb_(_fbb) {
    start_ = fbb_.StartTable();
  }
  BucketBuilder &operator=(const BucketBuilder &);
  flatbuffers::Offset<Bucket> Finish() {
    const auto end = fbb_.EndTable(start_, 2);
    auto o = flatbuffers::Offset<Bucket>(end);
    return o;
  }
};

inline flatbuffers::Offset<Bucket> CreateBucket(
    flatbuffers::FlatBufferBuilder &_fbb,
    flatbuffers::Offset<flatbuffers::String> bucketName = 0,
    flatbuffers::Offset<flatbuffers::String> alias = 0) {
  BucketBuilder builder_(_fbb);
  builder_.add_alias(alias);
  builder_.add_bucketName(bucketName);
  return builder_.Finish();
}

inline flatbuffers::Offset<Bucket> CreateBucketDirect(
    flatbuffers::FlatBufferBuilder &_fbb,
    const char *bucketName = nullptr,
    const char *alias = nullptr) {
  return CreateBucket(
      _fbb,
      bucketName ? _fbb.CreateString(bucketName) : 0,
      alias ? _fbb.CreateString(alias) : 0);
}

inline const flatbuf::cfg::Config *GetConfig(const void *buf) {
  return flatbuffers::GetRoot<flatbuf::cfg::Config>(buf);
}

inline bool VerifyConfigBuffer(
    flatbuffers::Verifier &verifier) {
  return verifier.VerifyBuffer<flatbuf::cfg::Config>(nullptr);
}

inline void FinishConfigBuffer(
    flatbuffers::FlatBufferBuilder &fbb,
    flatbuffers::Offset<flatbuf::cfg::Config> root) {
  fbb.Finish(root);
}

}  // namespace cfg
}  // namespace flatbuf

#endif  // FLATBUFFERS_GENERATED_CFGSCHEMA_FLATBUF_CFG_H_
