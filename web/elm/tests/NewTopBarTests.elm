module NewTopBarTests exposing (..)

import Dom
import Expect
import Html.Attributes as Attributes
import Test exposing (..)
import Test.Html.Query as Query
import Test.Html.Selector as THS exposing (tag, attribute, class)
import Test.Html.Event as Event
import NewTopBar exposing (MobileState(..))
import Task


smallScreen : NewTopBar.Model
smallScreen =
    let
        model =
            Tuple.first (NewTopBar.init True "")
    in
        { model | screenSize = NewTopBar.Mobile }


all : Test
all =
    describe "NewTopBarSearchInput"
        [ describe "on small screens"
            [ test "shows the search icon"
                (\_ ->
                    smallScreen
                        |> NewTopBar.view
                        |> Query.fromHtml
                        |> Query.findAll [ tag "button", class "search-btn" ]
                        |> Query.count (Expect.equal 1)
                )
            , test "shows the user info/logout button"
                (\_ ->
                    smallScreen
                        |> NewTopBar.view
                        |> Query.fromHtml
                        |> Query.findAll [ class "topbar-user-info" ]
                        |> Query.count (Expect.equal 1)
                )
            , test "shows no search input"
                (\_ ->
                    smallScreen
                        |> NewTopBar.view
                        |> Query.fromHtml
                        |> Query.findAll [ tag "input" ]
                        |> Query.count (Expect.equal 0)
                )
            , test "sends a ShowSearchInput message when the search button is clicked"
                (\_ ->
                    smallScreen
                        |> NewTopBar.view
                        |> Query.fromHtml
                        |> Query.find [ tag "button", class "search-btn" ]
                        |> Event.simulate Event.click
                        |> Event.expect NewTopBar.ShowSearchInput
                )
            , describe "on ShowSearchInput"
                [ test "hides the search button"
                    (\_ ->
                        smallScreen
                            |> NewTopBar.update NewTopBar.ShowSearchInput
                            |> Tuple.first
                            |> NewTopBar.view
                            |> Query.fromHtml
                            |> Query.findAll [ tag "button", class "search-btn" ]
                            |> Query.count (Expect.equal 0)
                    )
                , test "shows the search bar"
                    (\_ ->
                        smallScreen
                            |> NewTopBar.update NewTopBar.ShowSearchInput
                            |> Tuple.first
                            |> NewTopBar.view
                            |> Query.fromHtml
                            |> Query.findAll [ tag "input" ]
                            |> Query.count (Expect.equal 1)
                    )
                , test "hides the user info/logout button"
                    (\_ ->
                        smallScreen
                            |> NewTopBar.update NewTopBar.ShowSearchInput
                            |> Tuple.first
                            |> NewTopBar.view
                            |> Query.fromHtml
                            |> Query.findAll [ class "topbar-user-info" ]
                            |> Query.count (Expect.equal 0)
                    )
                , test "sends a BlurMsg message when the search input is blurred"
                    (\_ ->
                        smallScreen
                            |> NewTopBar.update NewTopBar.ShowSearchInput
                            |> Tuple.first
                            |> NewTopBar.view
                            |> Query.fromHtml
                            |> Query.find [ tag "input" ]
                            |> Event.simulate Event.blur
                            |> Event.expect NewTopBar.BlurMsg
                    )
                ]
            , describe "on BlurMsg"
                [ test "hides the search bar"
                    (\_ ->
                        smallScreen
                            |> NewTopBar.update NewTopBar.ShowSearchInput
                            |> Tuple.first
                            |> NewTopBar.update NewTopBar.BlurMsg
                            |> Tuple.first
                            |> NewTopBar.view
                            |> Query.fromHtml
                            |> Query.findAll [ tag "input" ]
                            |> Query.count (Expect.equal 0)
                    )
                , test "shows the search button"
                    (\_ ->
                        smallScreen
                            |> NewTopBar.update NewTopBar.ShowSearchInput
                            |> Tuple.first
                            |> NewTopBar.update NewTopBar.BlurMsg
                            |> Tuple.first
                            |> NewTopBar.view
                            |> Query.fromHtml
                            |> Query.findAll [ tag "button", class "search-btn" ]
                            |> Query.count (Expect.equal 1)
                    )
                , test "shows the user info/logout button"
                    (\_ ->
                        smallScreen
                            |> NewTopBar.update NewTopBar.ShowSearchInput
                            |> Tuple.first
                            |> NewTopBar.update NewTopBar.BlurMsg
                            |> Tuple.first
                            |> NewTopBar.view
                            |> Query.fromHtml
                            |> Query.findAll [ class "topbar-user-info" ]
                            |> Query.count (Expect.equal 1)
                    )
                ]
            ]
        , describe "on large screens"
            [ test "shows the entire search input on large screens"
                (\_ ->
                    NewTopBar.init True ""
                        |> Tuple.first
                        |> NewTopBar.view
                        |> Query.fromHtml
                        |> Query.find [ tag "input" ]
                        |> Query.has [ attribute (Attributes.placeholder "search") ]
                )
            , test "hides the search input on changing to a small screen"
                (\_ ->
                    NewTopBar.init True ""
                        |> Tuple.first
                        |> NewTopBar.update (NewTopBar.ScreenResized { width = 300, height = 800 })
                        |> Tuple.first
                        |> NewTopBar.view
                        |> Query.fromHtml
                        |> Query.findAll [ tag "input" ]
                        |> Query.count (Expect.equal 0)
                )
            ]
        ]
