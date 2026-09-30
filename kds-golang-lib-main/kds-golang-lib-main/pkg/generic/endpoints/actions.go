package endpoints

const YLA_GET_CLIENT_FAST = "getClientFast"
const YLA_GET_CLIENT_CONFIG = "getClientConfig"
const YLA_GET_USER_MENU = "getUserMenu"
const YLA_DO_USER_LOGIN = "doUserLogin"
const YLA_DO_USER_REGISTER = "doUserRegister"

const YLA_GET_USER_PAST_REDEMPTIONS = "getUserPastRedemptions"

//User Cards
const YLA_GET_USER_CARDS = "getUserCards"
const YLA_DO_ADD_USER_CARDS = "doAddUserCards"
const YLA_DO_REMOVE_USER_CARDS = "doRemoveUserCards"

//User Addresses
const YLA_GET_USER_ADDRESSES = "getUserAddresses"
const YLA_DO_ADD_USER_ADDRESSES = "doAddUserAddresses"
const YLA_DO_REMOVE_USER_ADDRESSES = "doRemoveUserAddresses"
const YLA_DO_USER_MAPPING = "doUserMapping"

// Payfast Tokens
const YLA_DO_TOKEN_CALLBACK = "doCardTokenCallback"
const YLA_DO_TOKEN_CHECK = "doCardTokenCheck"
const YLA_DO_PAYMENT_CALLBACK = "doCardPaymentCallback"
const YLA_DO_TOKEN_PAYMENT = "doCardTokenPayment"
const YLA_DO_CARD_PAYMENT_CALLBACK = "doPayfastCardPaymentCallback"

// Callbacks Lightspeed
const YLA_DO_V1_CALLBACK = "doV1LightSpeedCallback"
const YLA_DO_V3_CALLBACK = "doV3LightSpeedCallback"
const YLA_GET_ORDER_CALLBACKS = "getOrderCallbacks"
const YLA_DO_REALTIME_CALLBACKS = "doRealTimeCallbacks"

// Onesignal
const YLA_DO_UPDATE_ONESIGNAL = "doUpdateOnesignal"

// Loyalty
const LOYALTY_GET_USER = "getUserFromPOS"
const LOYALTY_GET_BUSINESS_OFFERS = "getBusinessOffers"
const LOYALTY_CLAIM_ITEM = "doClaimItem"
const LOYALTY_REVERSE_ITEM = "doReverseItem"
const LOYALTY_CLAIM_VOUCHER = "doClaimVoucher"

//YLA APPS
const YLA_GET_APPVERSION = "getAppversionYLA"
const YLA_DO_APPVERSION = "doAppversionYLA"

//YLA Orders
const YLA_DO_USER_ORDER_CREATE = "doUserOrderCreate"
const YLA_DO_CASH_ORDER_CREATE = "doCashOrderCreate"
const YLA_DO_USER_ORDER_ADD = "doUserOrderAdd"
const YLA_DO_USER_ORDER_DETAILS = "doUserOrderDetails"
const YLA_GET_USER_ORDER_FULL = "getUserOrderFull"
const YLA_DO_CONFIRM_USER_ORDER = "doConfirmUserOrder"
const YLA_GET_USER_ORDER_STATUS = "getUserOrderStatus"
const YLA_GET_USER_PAST_ORDERS = "getUserPastOrders"
const YLA_DO_USER_CANCEL_ORDER = "doUserCancelOrder"
const YLA_DO_USER_RETY_ORDER = "doUserRetryOrder"
const YLA_DO_ORDER_PAYMENT = "doOrderPayment"

// ZAPPER
const YLA_DO_ZAPPER_ITN = "doZapperITN"
const YLA_DO_ZAPPER_ITNWEB = "doZapperITNWeb"
const ZAPPER_DO_PAYMENT_FAILED = "doOrderPaymentWebFail"
const ZAPPER_DO_PAYMENT_SUCCESS = "doOrderPaymentWebSuccess"

const YLA_GET_USER_OFFERS = "getUserOffers"
const YLA_GET_USER_REDEMPTIONS = "getUserRedemptions"
