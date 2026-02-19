"""Full list of possible Exception enums."""

from enum import StrEnum


class ExceptionCodeEnum(StrEnum):
    """String enum of possible exceptions from the API."""

    TODO = "DevelopmentCodeWithRawExceptionOnly"

    # DP Get events
    DPGetEventsBadModelType = "DPGetEventsBadModelType"
    DPGetEventFailStatusCode = "DPGetEventFailStatusCode"
    DPGetEventNotMultipartForm = "DPGetEventNotMultipartForm"
    DPGetEventNotBoundaryFormData = "DPGetEventNotBoundaryFormData"
    DPGetEventsMissingResponseInfo = "DPGetEventsMissingResponseInfo"
    DPGetEventsFilepartMissingFromResponse = "DPGetEventsFilepartMissingFromResponse"
    DPGetEventBadResponseWithContent = "DPGetEventBadResponseWithContent"
    # DP Submit events
    DPSubmitEventsInvalidModel = "DPSubmitEventsInvalidModel"
    DPSubmitEventMessageTooLarge = "DPSubmitEventMessageTooLarge"
    DPSubmitEventsUnableToContactDP = "DPSubmitEventsUnableToContactDP"
    DPSubmitEventsUnableToSubmitEvents = "DPSubmitEventsUnableToSubmitEvents"
    DPSubmitEventsSubmittedEventsWereInvalid = "DPSubmitEventsSubmittedEventsWereInvalid"
    # DP Has binary
    DPHasBinaryUnableToRequestFile = "DPHasBinaryUnabletoRequestFile"
    DPHasBinaryNotFound = "DPHasBinaryNotFound"
    DPHasBinaryBadStatusCode = "DPHasBinaryBadStatusCode"
    # DP Get Binary
    DPGetBinaryOffsetTooLarge = "DPGetBinaryOffsetTooLarge"
    DPGetBinaryNotFound = "DPGetBinaryNotFound"
    DPGetBinaryBadStatusCode = "DPGetBinaryBadStatusCode"
    # DP Get Binary Async
    DPGetBinaryAsyncOffsetTooLarge = "DPGetBinaryAsyncOffsetTooLarge"
    DPGetBinaryAsyncNotFound = "DPGetBinaryAsyncNotFound"
    DPGetBinaryAsyncBadStatusCode = "DPGetBinaryAsyncBadStatusCode"
    DPAsyncGetBinaryUnexpectedError = "DPAsyncGetBinaryUnexpectedError"
    # Other
    ConvertingContentToAsyncIterable = "ConvertingContentToAsyncIterable"
    InvalidAsyncSubmissionStringBuffer = "InvalidAsyncSubmissionStringBuffer"
    InvalidAsyncSubmissionContentStream = " InvalidAsyncSubmissionContentStream"
    # DP Async Submit Binary
    DPAsyncSubmitBinaryUnableToContactDP = "DPAsyncSubmitBinaryUnableToContactDP"
    DPAsyncSubmitBinaryBadStatusCode = "DPAsyncSubmitBinaryBadStatusCode"
    DPAsyncSubmitBinaryInvalidResponseFormat = "DPAsyncSubmitBinaryInvalidResponseFormat"
    # DP Submit Binary
    DPSubmitBinaryInvalidSubmissionStringBuffer = "DPSubmitBinaryInvalidSubmissionStringBuffer"
    DPSubmitBinaryInvalidSubmissionContentStream = "DPSubmitBinaryInvalidSubmissionContentStream"
    # DP Unable to Submit binary
    DPSubmitBinaryUnableToContactDP = "DPSubmitBinaryUnableToContactDP"
    DPSubmitBinaryBadStatusCode = "DPSubmitBinaryBadStatusCode"
    DPSubmitBinaryInvalidResponseFormat = "DPSubmitBinaryInvalidResponseFormat"
    # DP Copy Binary
    DPCopyBinaryBadStatusCode = "DPCopyBinaryBadStatusCode"
    # DP Delete Binary
    DPDeleteBinaryBadStatusCode = "DPDeleteBinaryBadStatusCode"

    # Identify
    IdentifyDosIdentNotDos = "IdentifyDosIdentNotDos"
    NoFiletypeIdentificationHappened = "NoFiletypeIdentificationHappened"

    # models_network
    FeatureValueEncodingFailure = "FeatureValueEncodingFailure"
    FeatureValueDecodingFailure = "FeatureValueDecodingFailure"
    ConvertingStreamToInputEntityFailure = "ConvertingStreamToInputEntityFailure"

    # models_settings
    ConvertStringToDurationIncorrectNumberOfValuesAfterSplit = (
        "ConvertStringToDurationIncorrectNumberOfValuesAfterSplit"
    )
    ConvertStringToDurationInvalidDuration = "ConvertStringToDurationInvalidDuration"
    ConvertStringToDurationInvalidUnitProvided = "ConvertStringToDurationInvalidUnitProvided"

    ### Runner
    # test_template
    TestRunnerExecutionEventTooLarge = "TestRunnerExecutionEventTooLarge"

    ### Azul-metastore
    # context
    MetastoreContextBadSecurity = "MetastoreContextBadSecurity"
    MetastoreContextInsufficientPermissionsForWrite = "MetastoreContextInsufficientPermissionsForWrite"
    # entry
    MetastoreEntryBadInputParameters = "MetastoreEntryBadInputParameters"
    # ingestor
    MetastoreIngestorBadStatusDocument = "MetastoreIngestorBadStatusDocument"
    MetastoreIngestorGetDataNetworkError = "MetastoreIngestorGetDataNetworkError"
    # settings
    MetastoreSettingsPartitionNotSet = "MetastoreSettingsPartitionNotSet"
    MetastoreSettingsFieldsAreMissing = "MetastoreSettingsFieldsAreMissing"
    MetastoreSettingsExtraFieldsAreMissing = "MetastoreSettingsExtraFieldsAreMissing"
    # feature
    MetastoreFeatureEnrichmentFailed = "MetastoreFeatureEnrichmentFailed"
    # fileformat
    MetastoreFileFormatTooLargeForUnzip = "MetastoreFileFormatTooLargeForUnzip"
    MetastoreFileFormatTooManyFilesToExtract = "MetastoreFileFormatTooManyFilesToExtract"
    MetastoreFileFormatFileTooLargeToExtract = "MetastoreFileFormatFileTooLargeToExtract"
    MetastoreFileFormatFilePathTooLong = "MetastoreFileFormatFilePathTooLong"
    MetastoreFileFormatBadPathElevation = "MetastoreFileFormatBadPathElevation"
    MetastoreFileFormatNotAZipFile = "MetastoreFileFormatNotAZipFile"
    MetastoreFileFormatUnknownException = "MetastoreFileFormatUnknownException"
    MetastoreFileFormatZipFileRequiresPassword = "MetastoreFileFormatZipFileRequiresPassword"
    MetastoreFileFormatZipFileBadPasswordProvided = "MetastoreFileFormatZipFileBadPasswordProvided"
    MetastoreFileFormatNoFilesExtractedFromZip = "MetastoreFileFormatNoFilesExtractedFromZip"
    MetastoreFileFormatFileTooLargeForUnmalpz = "MetastoreFileFormatFileTooLargeForUnmalpz"
    # memcache
    MetastoreMemcacheTTLCacheAlreadyCreated = "MetastoreMemcacheTTLCacheAlreadyCreated"
    MetastoreMemcacheLRUCacheAlreadyCreated = "MetastoreMemcacheLRUCacheAlreadyCreated"
    # opensearch
    MetastoreOpensearchCantGetUserAccountInner = "MetastoreOpensearchCantGetUserAccountInner"
    MetastoreOpensearchCantGetUserAccount = "MetastoreOpensearchCantGetUserAccount"
    # search_data
    MetastoreSearchDataBadCredentials = "MetastoreSearchDataBadCredentials"
    MetastoreSearchDataMissingOrBadParameters = "MetastoreSearchDataMissingOrBadParameters"
    # search_query_parser
    MetastoreSearchQueryMissingToken = "MetastoreSearchQueryMissingToken"
    MetastoreSearchQueryInvalidUnescapeSequence = "MetastoreSearchQueryInvalidUnescapeSequence"
    MetastoreSearchQueryUnterminatedEscapeAtEnd = "MetastoreSearchQueryUnterminatedEscapeAtEnd"
    MetastoreSearchQueryNumberOfInputTokens = "MetastoreSearchQueryNumberOfInputTokens"
    MetastoreSearchQueryNumberExpressionNotInteger = "MetastoreSearchQueryNumberExpressionNotInteger"
    MetastoreSearchQueryStringFieldNotString = "MetastoreSearchQueryStringFieldNotString"
    MetastoreSearchQueryNumberExpressionUnexpectedTokenCount = (
        "MetastoreSearchQueryNumberExpressionUnexpectedTokenCount"
    )
    MetastoreSearchQueryInvalidType = "MetastoreSearchQueryInvalidType"
    # search_query
    MetastoreSearchQueryMissingContext = "MetastoreSearchQueryMissingContext"
    MetastoreSearchQueryTagSearchNotOnStringValue = "MetastoreSearchQueryTagSearchNotOnStringValue"
    MetastoreSearchQueryTagSearchNotWithEquals = "MetastoreSearchQueryTagSearchNotWithEquals"
    MetastoreSearchQueryTagDoesntExist = "MetastoreSearchQueryTagDoesntExist"
    MetastoreFeatureValueTagNotFound = "MetastoreFeatureValueTagNotFound"
    MetastoreSearchQueryUnreachableDuringConversion = "MetastoreSearchQueryUnreachableDuringConversion"
    MetastoreSearchQuerySearchForImplicitShouldBeLiteral = "MetastoreSearchQuerySearchForImplicitShouldBeLiteral"
    MetastoreSearchQueryAutocompleteParentNotATag = "MetastoreSearchQueryAutocompleteParentNotATag"
    MetastoreSearchQueryInvalidTermKey = "MetastoreSearchQueryInvalidTermKey"
    # string_filter
    MetastoreAiStringFilterFailure = "MetastoreAiStringFilterFailure"
    # tlsh
    MetastoreInvalidTLSHFormat = "MetastoreInvalidTLSHFormat"
    MetastoreInvalidTLSHLength = "MetastoreInvalidTLSHLength"
    # common/wrapper
    MetastoreOpensearchAuthFailure = "MetastoreOpensearchAuthFailure"
    MetastoreOpensearchFailedToCreateIndex = "MetastoreOpensearchFailedToCreateIndex"
    MetastoreOpensearchTemplateOldVersion = "MetastoreOpensearchTemplateOldVersion"
    MetastoreOpensearchKnnMisconfigured = "MetastoreOpensearchKnnMisconfigured"
    MetastoreOnlyAllowTopLevelBoolOrKnn = "MetastoreOnlyAllowTopLevelBoolOrKnn"
    MetastoreBadSecurityConversionExclude = "MetastoreBadSecurityConversionExclude"
    MetastoreBadSecurityConversionInclude = "MetastoreBadSecurityConversionInclude"
    MetastoreKnnTooManySearchTerms = "MetastoreKnnTooManySearchTerms"
    MetastoreUnknownDocType = "MetastoreUnknownDocType"
    # annotation
    MetastoreUnknownAnnotation = "MetastoreUnknownAnnotation"
    MetastoreAnnotationBadCharacterInTag = "MetastoreAnnotationBadCharacterInTag"
    MetastoreAnnotationTagTooLong = "MetastoreAnnotationTagTooLong"
    MetastoreAnnotationCommentTooLong = "MetastoreAnnotationCommentTooLong"
    # base_encoders
    MetastoreEncoderInvalidTimestamp = "MetastoreEncoderInvalidTimestamp"
    MetastoreInvalidPartitionFormatUnits = "MetastoreInvalidPartitionFormatUnits"
    # binary2
    MetastoreEncodeErrorMissingTrackingInfo = "MetastoreEncodeErrorMissingTrackingInfo"
    MetastoreEncodingMissingSource = "MetastoreEncodingMissingSource"
    MetastoreEncodingInvalidSSDeep = "MetastoreEncodingInvalidSSDeep"
    # basic_events
    MetastorePreProcessTooManyStreams = "MetastorePreProcessTooManyStreams"
    MetastorePreProcessTooManyFeatures = "MetastorePreProcessTooManyFeatures"
    MetastoreEncoderInvalidPluginConfig = "MetastoreEncoderInvalidPluginConfig"
    # age_off
    MetastoreFailedToAgeoffDocs = "MetastoreFailedToAgeoffDocs"
    # cache
    MetastoreCacheTooManyIds = "MetastoreCacheTooManyIds"
    # purge
    MetastoreCannotCreatePurgeFolder = "MetastoreCannotCreatePurgeFolder"
    MetastoreMetaDataDeletionFailure = "MetastoreMetaDataDeletionFailure"
    MetastoreNothingToPurge = "MetastoreNothingToPurge"
    # binary_event
    MetastoreBadMappingToOpensearch = "MetastoreBadMappingToOpensearch"
    # binary_feature
    MetastoreInvalidAfterProvided = "MetastoreInvalidAfterProvided"
    MetastoreFailedToParseTermQuery = "MetastoreFailedToParseTermQuery"
    MetastoreSha256NotProvidedForFindFamily = "MetastoreSha256NotProvidedForFindFamily"
    # binary_find
    MetastoreBinaryFindTooManyBinariesRequested = "MetastoreBinaryFindTooManyBinariesRequested"
    MetastoreBinaryFindFaildToParseSearchTerm = "MetastoreBinaryFindFaildToParseSearchTerm"
    # binary_read
    MetastoreSha256NotProvidedForFindingStreamRefs = "MetastoreSha256NotProvidedForFindingStreamRefs"
    # binary_similar
    MetastoreLibFuzzyFailedToInitalise = "MetastoreLibFuzzyFailedToInitalise"
    MetastoreReadSimilarSSDeepBadFuzzyHash = "MetastoreReadSimilarSSDeepBadFuzzyHash"
    # binary_submit_manual
    MetastoreDispatcherRejectedEvents = "MetastoreDispatcherRejectedEvents"
    MetastoreSubmissionsCantCreateInsertionEvents = "MetastoreSubmissionsCantCreateInsertionEvents"
    # binary_submit
    MetastoreNoSourcesForBinarySubmission = "MetastoreNoSourcesForBinarySubmission"
    MetastoreBadAugmentedStreamLabel = "MetastoreBadAugmentedStreamLabel"
    MetastoreInvalidSourceForBinarySubmission = "MetastoreInvalidSourceForBinarySubmission"
    MetastoreBadSourceReferenceDefinition = "MetastoreBadSourceReferenceDefinition"
    MetastoreUnableToSubmitBinaryEventImmediately = "MetastoreUnableToSubmitBinaryEventImmediately"
    MetastoreBinarySubmitDispatcherRejectedEvent = "MetastoreBinarySubmitDispatcherRejectedEvent"
    MetastoreBadBinarySubmissionBadSecurityString = "MetastoreBadBinarySubmissionBadSecurityString"
    MetastoreBadBinarySubmissionUserSecurity = "MetastoreBadBinarySubmissionUserSecurity"
    MetastoreBadBinarySubmissionSourceOrParent = "MetastoreBadBinarySubmissionSourceOrParent"
    MetastoreBadBinarySubmissionParentAndSource = "MetastoreBadBinarySubmissionParentAndSource"
    MetastoreBadBinarySubmissionNoSha256Provided = "MetastoreBadBinarySubmissionNoSha256Provided"
    MetastoreBadBinarySubmissionParentNotFound = "MetastoreBadBinarySubmissionParentNotFound"
    MetastoreUnableToExtractProvidedArchive = "MetastoreUnableToExtractProvidedArchive"
    MetastoreDatalessSubmissionBinaryDoesNotExist = "MetastoreDatalessSubmissionBinaryDoesNotExist"
    MetastoreUnableToExtractAnyFiles = "MetastoreUnableToExtractAnyFiles"
    # binaries_data
    MetastoreBinaryNotFound = "MetastoreCheckedBinaryNotFound"
    MetastoreBinaryStreamNotFound = "MetastoreBinaryStreamNotFound"
    MetastoreInvalidSha256Provided = "MetastoreInvalidSha256Provided"
    MetastoreBulkUnableToDownloadAnyBinaries = "MetastoreBulkUnableToDownloadAnyBinaries"
    MetastoreNoBinariesDownloaded = "MetastoreNoBinariesDownloaded"
    MetastoreDownloadingBadStreamType = "MetastoreDownloadingBadStreamType"
    MetastoreInvalidStringsRegexProvided = "MetastoreInvalidStringsRegexProvided"
    MetastoreInvalidHexPatternProvided = "MetastoreInvalidHexPatternProvided"
    # binaries_submit
    MetastoreCannotParseTimestampToUTC = "MetastoreCannotParseTimestampToUTC"
    MetastoreInvalidJson = "MetastoreInvalidJson"
    MetastoreAltStreamsLabelsDoesNotMatch = "MetastoreAltStreamsLabelsDoesNotMatch"
    # binaries
    MetastoreFindBinaryInvalidSearch = "MetastoreFindBinaryInvalidSearch"
    MetastorePotentiallyInvalidQueryOption = "MetastorePotentiallyInvalidQueryOption"
    MetastoreIncludeCousinsInvalidEnum = "MetastoreIncludeCousinsInvalidEnum"
    MetastoreTagBinaryBadSecurityString = "MetastoreTagBinaryBadSecurityString"
    MetastoreTagBinaryInvalidSecurity = "MetastoreTagBinaryInvalidSecurity"
    MetastoreInvalidAnnotationForCreate = "MetastoreInvalidAnnotationForCreate"
    MetastoreCantDeleteTagFromBinary = "MetastoreCantDeleteTagFromBinary"
    # features
    MetastoreInvalidDeleteTag = "MetastoreInvalidDeleteTag"
    MetastoreNoFeatureValuesFound = "MetastoreNoFeatureValuesFound"
    MetastoreFeaturesInvalidPivotFeatures = "MetastoreFeaturesInvalidPivotFeatures"
    # plugins
    MetastoreNoPluginsInAzul = "MetastoreNoPluginsInAzul"
    MetastoreNoPluginStatusesInAzul = "MetastoreNoPluginStatusesInAzul"
    MetastorePluginNotInAzul = "MetastorePluginNotInAzul"
    # purge
    MetastoreUserNotAllowedToPurge = "MetastoreUserNotAllowedToPurge"
    MetastoreInvalidTimestampForPurge = "MetastoreInvalidTimestampForPurge"
    MetastoreInvalidPurgeExceptionApi = "MetastoreInvalidPurgeExceptionApi"
    # quick
    MetastoreUserUnauthorized = "MetastoreUserUnauthorized"
    MetastoreSetSecurityHeaderUnexpected = "MetastoreSetSecurityHeaderUnexpected"
    MetastoreUserInfoNotAvailable = "MetastoreUserInfoNotAvailable"
    # sources
    MetastoreNoSourcesInAzul = "MetastoreNoSourcesInAzul"
    MetastoreSourceNotFound = "MetastoreSourceNotFound"
    MetastoreSourceNoReferences = "MetastoreSourceNoReferences"
    MetastoreSourceSubmissionNoInformationFound = "MetastoreSourceSubmissionNoInformationFound"

    ### Security
    # friendly
    SecurityMissingOrigin = "SecurityMissingOrigin"
    SecurityClassificationDoesntSupportReleasability = "SecurityClassificationDoesntSupportReleasability"
    SecurityInvalidReleasabilityGroup = "SecurityInvalidReleasabilityGroup"
    SecurityInvalidExclusiveGroup = "SecurityInvalidExclusiveGroup"
    SecurityInvalidInclusiveGroup = "SecurityInvalidInclusiveGroup"
    SecurityInvalidMarkingsGroup = "SecurityInvalidMarkingsGroup"
    SecurityNoClassificationInRawSecurity = "SecurityNoClassificationInRawSecurity"
    SecurityInvalidGroupsWhileNormalising = "SecurityInvalidGroupsWhileNormalising"
    SecurityInvalidSecurityStringWhileNormalising = "SecurityInvalidSecurityStringWhileNormalising"
    SecurityInvalidReleasabilitiesConvertFromLabels = "SecurityInvalidReleasabilitiesConvertFromLabels"
    SecurityInvalidLabelConvertingFromLabels = "SecurityInvalidLabelConvertingFromLabels"

    # security
    SecurityMinRequiredAccessNotFound = "SecurityMinRequiredAccessNotFound"
    SecuritySecurityDefaultNotSet = "SecuritySecurityDefaultNotSet"
    SecurityNoCommonSecurityUnviewable = "SecurityNoCommonSecurityUnviewable"
    SecurityUserCannotAccessExclusive = "SecurityUserCannotAccessExclusive"
    SecurityUserCannotAccessInclusive = "SecurityUserCannotAccessInclusive"
    SecurityUserCannotAccessMarkings = "SecurityUserCannotAccessMarkings"
    SecurityUnmatchedLabelsGoingSafeToUnsafe = "SecurityUnmatchedLabelsGoingSafeToUnsafe"
    SecurityUnmatchedLabelsGoingUnsafeToSafe = "SecurityUnmatchedLabelsGoingUnsafeToSafe"
    SecurityUserDoesNotHaveMinimumAccess = "SecurityUserDoesNotHaveMinimumAccess"

    # settings
    SecurityConfigLabelsWithExtraSpaces = "SecurityConfigLabelsWithExtraSpaces"
    SecurityConfigReleasabilitiesMissingRequiredPrefix = "SecurityConfigReleasabilitiesMissingRequiredPrefix"
    SecurityConfigLabelDefinedTwice = "SecurityConfigLabelDefinedTwice"
    SecurityConfigGroupLabelMustNotHaveSpaces = "SecurityConfigGroupLabelMustNotHaveSpaces"
    SecurityConfigMultipleValuesMappedToSameValue = "SecurityConfigMultipleValuesMappedToSameValue"

    # restapi
    SecurityNormaliseInvalidSecurity = "SecurityNormaliseInvalidSecurity"
    SecurityInvalidMaxSecurity = "SecurityInvalidMaxSecurity"
    SecurityEmptyResultForMaxSecurity = "SecurityEmptyResultForMaxSecurity"
    SecurityUserInfoCannotBeAcquired = "SecurityUserInfoCannotBeAcquired"

    ### azul-restapi-server
    # pat
    RestapiAllowedPATAction = "RestapiAllowedPATAction"
