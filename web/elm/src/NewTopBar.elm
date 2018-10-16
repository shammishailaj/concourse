module NewTopBar
    exposing
        ( Model
        , ScreenSize(..)
        , Msg(FilterMsg, KeyDown, LoggedOut, UserFetched, ShowSearchInput, BlurMsg, Noop, ScreenResized)
        , MobileState(..)
        , fetchUser
        , init
        , update
        , view
        )

import Array
import Concourse
import Concourse.Team
import Concourse.User
import Dom
import Html exposing (Html)
import Html.Attributes exposing (class, classList, href, id, placeholder, src, type_, value)
import Html.Events exposing (..)
import Http
import Keyboard
import LoginRedirect
import Navigation
import QueryString
import RemoteData exposing (RemoteData)
import Task
import TopBar exposing (userDisplayName)
import UserState exposing (UserState(..))
import Window


type alias Model =
    { teams : RemoteData.WebData (List Concourse.Team)
    , userState : UserState
    , userMenuVisible : Bool
    , query : String
    , showSearch : Bool
    , showAutocomplete : Bool
    , selectionMade : Bool
    , selection : Int
    , screenSize : ScreenSize
    , mobileSearchState : MobileState
    }


type ScreenSize
    = Mobile
    | Desktop


type MobileState
    = Expanded
    | Collapsed


type Msg
    = Noop
    | UserFetched (RemoteData.WebData Concourse.User)
    | TeamsFetched (RemoteData.WebData (List Concourse.Team))
    | LogIn
    | LogOut
    | LoggedOut (Result Http.Error ())
    | FilterMsg String
    | FocusMsg
    | BlurMsg
    | SelectMsg Int
    | KeyDown Keyboard.KeyCode
    | ToggleUserMenu
    | ShowSearchInput
    | ScreenResized Window.Size


init : Bool -> String -> ( Model, Cmd Msg )
init showSearch query =
    ( { teams = RemoteData.Loading
      , userState = UserStateUnknown
      , userMenuVisible = False
      , query = query
      , showSearch = showSearch
      , showAutocomplete = False
      , selectionMade = False
      , selection = 0
      , screenSize = Desktop
      , mobileSearchState = Collapsed
      }
    , Cmd.batch
        [ fetchUser
        , fetchTeams
        , Task.perform ScreenResized Window.size
        ]
    )


getScreenSize : Window.Size -> ScreenSize
getScreenSize size =
    if size.width < 812 then
        Mobile
    else
        Desktop


queryStringFromSearch : String -> String
queryStringFromSearch query =
    case query of
        "" ->
            QueryString.render QueryString.empty

        query ->
            QueryString.render <|
                QueryString.add "search" query QueryString.empty


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        Noop ->
            ( model, Cmd.none )

        FilterMsg query ->
            ( { model | query = query }
            , Cmd.batch
                [ Task.attempt (always Noop) (Dom.focus "search-input-field")
                , Navigation.modifyUrl (queryStringFromSearch query)
                ]
            )

        UserFetched user ->
            case user of
                RemoteData.Success user ->
                    ( { model | userState = UserStateLoggedIn user }, Cmd.none )

                _ ->
                    ( { model | userState = UserStateLoggedOut }
                    , Cmd.none
                    )

        LogIn ->
            ( model
            , LoginRedirect.requestLoginRedirect ""
            )

        LogOut ->
            ( model, logOut )

        LoggedOut (Ok _) ->
            let
                redirectUrl =
                    case model.showSearch of
                        True ->
                            "/dashboard"

                        False ->
                            "/dashboard/hd"
            in
                ( { model
                    | userState = UserStateLoggedOut
                    , userMenuVisible = False
                    , teams = RemoteData.Loading
                  }
                , Navigation.newUrl redirectUrl
                )

        LoggedOut (Err err) ->
            flip always (Debug.log "failed to log out" err) <|
                ( model, Cmd.none )

        ToggleUserMenu ->
            ( { model | userMenuVisible = not model.userMenuVisible }, Cmd.none )

        TeamsFetched response ->
            ( { model | teams = response }, Cmd.none )

        FocusMsg ->
            ( { model | showAutocomplete = True }, Cmd.none )

        BlurMsg ->
            let
                newModel =
                    hideSearchInput model
            in
                ( { newModel | showAutocomplete = False, selectionMade = False, selection = 0 }, Cmd.none )

        SelectMsg index ->
            ( { model | selectionMade = True, selection = index + 1 }, Cmd.none )

        KeyDown keycode ->
            if not model.showAutocomplete then
                ( { model | selectionMade = False, selection = 0 }, Cmd.none )
            else
                case keycode of
                    -- enter key
                    13 ->
                        if not model.selectionMade then
                            ( model, Cmd.none )
                        else
                            let
                                options =
                                    Array.fromList (autocompleteOptions model)

                                index =
                                    (model.selection - 1) % Array.length options

                                selectedItem =
                                    case Array.get index options of
                                        Nothing ->
                                            model.query

                                        Just item ->
                                            item
                            in
                                ( { model | selectionMade = False, selection = 0, query = selectedItem }, Cmd.none )

                    -- up arrow
                    38 ->
                        ( { model | selectionMade = True, selection = model.selection - 1 }, Cmd.none )

                    -- down arrow
                    40 ->
                        ( { model | selectionMade = True, selection = model.selection + 1 }, Cmd.none )

                    -- escape key
                    27 ->
                        ( model, Task.attempt (always Noop) (Dom.blur "search-input-field") )

                    _ ->
                        ( { model | selectionMade = False, selection = 0 }, Cmd.none )

        ShowSearchInput ->
            showSearchInput model

        ScreenResized size ->
            ( { model | screenSize = getScreenSize size }, Cmd.none )


showSearchInput : Model -> ( Model, Cmd Msg )
showSearchInput model =
    let
        newModel =
            { model | mobileSearchState = Expanded }
    in
        case model.screenSize of
            Mobile ->
                ( newModel, Task.attempt (always Noop) (Dom.focus "search-input-field") )

            Desktop ->
                ( newModel, Cmd.none )


hideSearchInput : Model -> Model
hideSearchInput model =
    { model | mobileSearchState = Collapsed }


viewUserState : UserState -> Bool -> Html Msg
viewUserState userState userMenuVisible =
    case userState of
        UserStateUnknown ->
            Html.text ""

        UserStateLoggedOut ->
            Html.div [ class "user-id", onClick LogIn ]
                [ Html.a
                    [ href "/sky/login"
                    , Html.Attributes.attribute "aria-label" "Log In"
                    , class "login-button"
                    ]
                    [ Html.text "login"
                    ]
                ]

        UserStateLoggedIn user ->
            Html.div [ class "user-info" ]
                [ Html.div [ class "user-id", onClick ToggleUserMenu ]
                    [ Html.text <|
                        userDisplayName user
                    ]
                , Html.div [ classList [ ( "user-menu", True ), ( "hidden", not userMenuVisible ) ], onClick LogOut ]
                    [ Html.a
                        [ Html.Attributes.attribute "aria-label" "Log Out"
                        ]
                        [ Html.text "logout"
                        ]
                    ]
                ]


searchBar : Model -> Html Msg
searchBar model =
    Html.div [ class "topbar-search-form" ]
        [ Html.input
            [ class "search-input-field"
            , id "search-input-field"
            , type_ "text"
            , placeholder "search"
            , onInput FilterMsg
            , onFocus FocusMsg
            , onBlur BlurMsg
            , value model.query
            ]
            []
        , Html.span
            [ classList [ ( "search-clear-button", True ), ( "active", not <| String.isEmpty model.query ) ]
            , id "search-clear-button"
            , onClick (FilterMsg "")
            ]
            []
        ]


view : Model -> Html Msg
view model =
    Html.div [ class "module-topbar" ]
        [ Html.div [ class "topbar-logo" ] [ Html.a [ class "logo-image-link", href "#" ] [] ]
        , Html.div [ class "topbar-login" ]
            (case model.screenSize of
                Desktop ->
                    [ Html.div [ class "topbar-user-info" ]
                        [ viewUserState model.userState model.userMenuVisible
                        ]
                    ]

                Mobile ->
                    case model.mobileSearchState of
                        Collapsed ->
                            [ Html.div [ class "topbar-user-info" ]
                                [ viewUserState model.userState model.userMenuVisible
                                ]
                            ]

                        Expanded ->
                            []
            )
        , Html.div [ classList [ ( "topbar-search", True ), ( "hidden", not model.showSearch ) ] ]
            [ (case model.screenSize of
                Desktop ->
                    searchBar model

                Mobile ->
                    case model.mobileSearchState of
                        Collapsed ->
                            Html.button
                                [ class "search-btn"
                                , onClick ShowSearchInput
                                ]
                                []

                        Expanded ->
                            searchBar model
              )
            , Html.ul [ classList [ ( "hidden", not model.showAutocomplete ), ( "search-options", True ) ] ] <|
                let
                    options =
                        autocompleteOptions model
                in
                    List.indexedMap
                        (\index option ->
                            Html.li
                                [ classList
                                    [ ( "search-option", True )
                                    , ( "active", model.selectionMade && index == (model.selection - 1) % List.length options )
                                    ]
                                , onMouseDown (FilterMsg option)
                                , onMouseOver (SelectMsg index)
                                ]
                                [ Html.text option ]
                        )
                        options
            ]
        ]


fetchUser : Cmd Msg
fetchUser =
    Cmd.map UserFetched <|
        RemoteData.asCmd Concourse.User.fetchUser


fetchTeams : Cmd Msg
fetchTeams =
    Cmd.map TeamsFetched <|
        RemoteData.asCmd Concourse.Team.fetchTeams


autocompleteOptions : Model -> List String
autocompleteOptions model =
    case String.trim model.query of
        "" ->
            [ "status: ", "team: " ]

        "status:" ->
            [ "status: paused", "status: pending", "status: failed", "status: errored", "status: aborted", "status: running", "status: succeeded" ]

        "team:" ->
            case model.teams of
                RemoteData.Success teams ->
                    List.map (\team -> "team: " ++ team.name) <| List.take 10 teams

                _ ->
                    []

        _ ->
            []


logOut : Cmd Msg
logOut =
    Task.attempt LoggedOut Concourse.User.logOut
