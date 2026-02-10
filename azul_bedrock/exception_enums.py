"""Full list of possible Exception enums."""

from enum import StrEnum


class ExceptionCodeEnum(StrEnum):
    """String enum of possible exceptions from the API."""

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
